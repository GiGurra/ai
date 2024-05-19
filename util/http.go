package util

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
)

var HttpClient = resty.New()

type GetParams struct {
	QueryParams map[string]string
	Headers     map[string]string
	OkStatusFn  func(statusCode int) bool
}

type StatusCodeError struct {
	StatusCode int
}

func (e StatusCodeError) Error() string {
	return fmt.Sprintf("unexpected status code: %d", e.StatusCode)
}

type MissingResponseBody struct {
}

func (e MissingResponseBody) Error() string {
	return "missing response body"
}

type FailedToParseResponse struct {
	Err error
}

func (e FailedToParseResponse) Error() string {
	return fmt.Sprintf("failed to parse response: %v", e.Err)
}

func HttpGetJson[T any](url string, params GetParams) (T, error) {

	request := HttpClient.R()

	okStatusFn := func(statusCode int) bool {
		if params.OkStatusFn != nil {
			return params.OkStatusFn(statusCode)
		} else {
			return statusCode == 200 // only 200, since we are expecting a json response
		}
	}

	if len(params.QueryParams) > 0 {
		request.SetQueryParams(params.QueryParams)
	}

	if len(params.Headers) > 0 {
		request.SetHeaders(params.Headers)
	}

	var zero T

	res, err := request.Get(url)
	if err != nil {
		return zero, err
	}

	if !okStatusFn(res.StatusCode()) {
		return zero, StatusCodeError{StatusCode: res.StatusCode()}
	}

	if len(res.Body()) == 0 {
		return zero, MissingResponseBody{}
	}

	var result T
	err = json.Unmarshal(res.Body(), &result)
	if err != nil {
		return zero, FailedToParseResponse{Err: err}
	}

	return result, err
}
