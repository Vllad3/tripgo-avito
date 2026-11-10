package middleware

import "net/http"

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func observed(next http.Handler, w http.ResponseWriter, r *http.Request, finish func(status int)) {
	sw := &statusWriter{ResponseWriter: w}

	defer func() {
		if rec := recover(); rec != nil {
			finish(http.StatusInternalServerError)
			panic(rec)
		}

		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}
		finish(status)
	}()

	next.ServeHTTP(sw, r)
}
