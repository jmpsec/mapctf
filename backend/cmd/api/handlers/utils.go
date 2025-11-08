package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"

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
const errorContent = "❌"
const okContent = "✅"

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
