package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	perTeamThrottleCapacityExceeded = "Server capacity exceeded."
	perTeamThrottleTimedOut         = "Timed out while waiting for a pending request to complete."
	perTeamThrottleContextCanceled  = "Context was canceled."
)

type perTeamThrottleState struct {
	tokens        chan struct{}
	backlogTokens chan struct{}
}

func newPerTeamThrottleState(limit, backlogLimit int) *perTeamThrottleState {
	state := &perTeamThrottleState{
		tokens:        make(chan struct{}, limit),
		backlogTokens: make(chan struct{}, limit+backlogLimit),
	}

	for i := 0; i < limit+backlogLimit; i++ {
		if i < limit {
			state.tokens <- struct{}{}
		}
		state.backlogTokens <- struct{}{}
	}

	return state
}

func (h *HandlersMap) PerTeamThrottleBacklog(limit, backlogLimit int, backlogTimeout time.Duration) func(http.Handler) http.Handler {
	if limit < 1 {
		panic("mapctf/handlers: PerTeamThrottleBacklog expects limit > 0")
	}
	if backlogLimit < 0 {
		panic("mapctf/handlers: PerTeamThrottleBacklog expects backlogLimit >= 0")
	}

	var (
		mu     sync.Mutex
		states = make(map[string]*perTeamThrottleState)
	)

	getState := func(key string) *perTeamThrottleState {
		mu.Lock()
		defer mu.Unlock()

		state, ok := states[key]
		if ok {
			return state
		}

		state = newPerTeamThrottleState(limit, backlogLimit)
		states[key] = state
		return state
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, err := h.teamThrottleKey(r)
			if err != nil {
				http.Error(w, "Unable to resolve team rate limit key.", http.StatusInternalServerError)
				return
			}

			state := getState(key)
			ctx := r.Context()

			select {
			case <-ctx.Done():
				http.Error(w, perTeamThrottleContextCanceled, http.StatusTooManyRequests)
				return
			case backlogToken := <-state.backlogTokens:
				defer func() {
					state.backlogTokens <- backlogToken
				}()

				select {
				case processingToken := <-state.tokens:
					defer func() {
						state.tokens <- processingToken
					}()
					next.ServeHTTP(w, r)
					return
				default:
				}

				timer := time.NewTimer(backlogTimeout)
				defer timer.Stop()

				select {
				case <-timer.C:
					w.Header().Set("Retry-After", strconv.Itoa(int(backlogTimeout.Seconds())))
					http.Error(w, perTeamThrottleTimedOut, http.StatusTooManyRequests)
					return
				case <-ctx.Done():
					http.Error(w, perTeamThrottleContextCanceled, http.StatusTooManyRequests)
					return
				case processingToken := <-state.tokens:
					defer func() {
						state.tokens <- processingToken
					}()
					next.ServeHTTP(w, r)
					return
				}
			default:
				w.Header().Set("Retry-After", strconv.Itoa(int(backlogTimeout.Seconds())))
				http.Error(w, perTeamThrottleCapacityExceeded, http.StatusTooManyRequests)
				return
			}
		})
	}
}

func (h *HandlersMap) teamThrottleKey(r *http.Request) (string, error) {
	if h.Sessions == nil || h.Users == nil {
		return "", errors.New("sessions or users manager not initialized")
	}

	uuid := h.Config.Map.UUID
	username := h.Sessions.GetString(r.Context(), string(ContextKeyUser))
	if username == "" {
		return "", errors.New("user not authenticated")
	}

	user, err := h.Users.Get(username, uuid)
	if err != nil {
		return "", err
	}

	if user.TeamID == 0 {
		return "user:" + username, nil
	}

	return "team:" + strconv.FormatUint(uint64(user.TeamID), 10), nil
}
