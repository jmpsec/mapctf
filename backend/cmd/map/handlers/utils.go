package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ContentType for header key
const ContentType string = "Content-Type"

// JSONApplication for Content-Type headers
const JSONApplication string = "application/json"

// JSONApplicationUTF8 for Content-Type headers, UTF charset
const JSONApplicationUTF8 string = JSONApplication + "; charset=UTF-8"

// Default content
const rootContent = "🌍"
const errorContent = "❌"
const okContent = "✅"
const forbiddenContent = "🚫"

// DebugHTTP - Helper for debugging purposes and dump a full HTTP request
func DebugHTTP(r *http.Request, showBody bool) string {
	var debug string
	debug = fmt.Sprintf("%s\n", "---------------- request")
	requestDump, err := httputil.DumpRequest(r, showBody)
	if err != nil {
		log.Err(err).Msg("error while dumprequest")
	}
	debug += fmt.Sprintf("%s\n", string(requestDump))
	if !showBody {
		debug += fmt.Sprintf("%s\n", "---------------- No Body")
	}
	debug += fmt.Sprintf("%s\n", "---------------- end")
	return debug
}

// DebugHTTPDump - Helper for debugging purposes and dump a full HTTP request
func DebugHTTPDump(l *zerolog.Logger, r *http.Request, showBody bool) {
	l.Log().Msg(DebugHTTP(r, showBody))
}

// HTTPResponse - Helper to send HTTP response
func HTTPResponse(w http.ResponseWriter, cType string, code int, data interface{}) {
	if cType != "" {
		w.Header().Set(ContentType, cType)
	}
	// Serialize if is not a []byte
	var content []byte
	if x, ok := data.([]byte); ok {
		content = x
	} else {
		var err error
		content, err = json.Marshal(data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errStr := "error serializing response"
			log.Err(err).Msg(errStr)
			content = []byte(errStr)
		}
	}
	w.WriteHeader(code)
	_, _ = w.Write(content)
}

// getRealIP extracts the real client IP from the request
// Works with chi's middleware.RealIP middleware which processes X-Real-Ip and X-Forwarded-For headers
func getRealIP(r *http.Request) string {
	// Check X-Real-Ip header (set by nginx, load balancers, etc.)
	if ip := r.Header.Get("X-Real-Ip"); ip != "" {
		return strings.TrimSpace(ip)
	}
	// Check X-Forwarded-For header (can contain multiple IPs: "client, proxy1, proxy2")
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, the first one is the original client IP
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	// Fallback to RemoteAddr (already processed by chi's RealIP middleware)
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
