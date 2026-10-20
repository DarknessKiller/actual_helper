package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const sessionCookieName = "ah_sid"
const sessionIDBytes = 32

// sessionID returns the caller's session id from the ah_sid cookie, or "".
func sessionID(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return ""
	}
	return cookie.Value
}

// ensureSession validates an existing session id or issues a new one, setting
// the cookie on the response. Session rules only ever apply to the holder of
// this cookie, so tampering costs the attacker nothing.
func ensureSession(w http.ResponseWriter, r *http.Request) (string, error) {
	if sid := sessionID(r); validSessionID(sid) {
		return sid, nil
	}
	buf := make([]byte, sessionIDBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	sid := hex.EncodeToString(buf)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sid,
		Path:     "/",
		MaxAge:   int(24 * 3600),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secureCookie(r),
	})
	return sid, nil
}

// validSessionID checks the id shape (64 hex chars) before it is used as a map key.
func validSessionID(sid string) bool {
	if len(sid) != sessionIDBytes*2 {
		return false
	}
	for _, c := range sid {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// secureCookie flags the Secure attribute when the request is already HTTPS
// (behind a reverse proxy) so local http testing still works.
func secureCookie(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return r.Header.Get("X-Forwarded-Proto") == "https"
}
