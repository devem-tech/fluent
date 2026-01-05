package fluent

import (
	"encoding/json"
	"io"
	"net/http"
)

// Response обёртка над http.Response и ошибкой, полученной при выполнении запроса.
type Response struct {
	resp *http.Response
	err  error
}

// Raw читает и возвращает весь ответ сервера как []byte.
// Если при запросе или чтении возникла ошибка — возвращает ошибку.
func (r *Response) Raw() ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	defer r.resp.Body.Close()

	return io.ReadAll(r.resp.Body)
}

// Body возвращает io.ReadCloser для тела ответа.
// Вызовите r.Body().Close() самостоятельно, если читаете тело вручную.
// Если при запросе возникла ошибка — возвращает ошибку.
func (r *Response) Body() (io.ReadCloser, error) {
	if r.err != nil {
		return nil, r.err
	}

	return r.resp.Body, nil
}

// Error возвращает ошибку, возникшую при выполнении HTTP-запроса.
// Если ошибки не было — возвращает nil.
func (r *Response) Error() error {
	return r.err
}

// Into декодирует тело ответа из JSON в структуру типа T.
// Возвращает значение T и ошибку, если она возникла.
// Тело ответа автоматически закрывается.
func Into[T any](r *Response) (T, error) {
	var res T

	if r.err != nil {
		return res, r.err
	}
	defer r.resp.Body.Close()

	err := json.NewDecoder(r.resp.Body).Decode(&res)

	return res, err
}
