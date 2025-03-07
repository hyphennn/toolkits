// Package httputils
// Author: hyphen
// Copyright 2022 hyphen. All rights reserved.
// Create-time: 2022/7/18
package httputils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
)

var client = &http.Client{}

func AccessBySystemCall[T any](ctx context.Context, url string, method string, header map[string]string,
	param map[string]string, body any) (T, error) {
	var ret T
	sb := bytes.Buffer{}
	sb.WriteString(url)
	sb.WriteByte('?')
	for k, v := range param {
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(v)
		sb.WriteByte('&')
	}
	args := []string{"--location", "--request", method, sb.String()}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return ret, err
	}
	if len(bodyJSON) != 0 {
		args = append(args, []string{"--data-raw", `'` + string(bodyJSON) + `''`}...)
	}

	if header != nil {
		for k, v := range header {
			args = append(args, []string{`--header`, `'` + k + `: ` + v + `'`}...)
		}
	}

	cmd := exec.Command("curl", args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	slog.InfoContext(ctx, "[AccessBySystemCall]prepared to execute: %s", cmd.String())
	err = cmd.Run()
	if err != nil {
		return ret, fmt.Errorf("[AccessBySystemCall]exec cmd failed: %w", err)
	}
	respBody := stdout.Bytes()
	if len(respBody) != 0 {
		err = json.Unmarshal(respBody, &ret)
		if err != nil {
			return ret, fmt.Errorf("[AccessBySystemCall]unmarshal stdout failed: %w", err)
		}
	}
	return ret, nil
}

func AccessResp(ctx context.Context, url string, method string, header map[string]string, param map[string]string,
	body any, setAuthorization func(*http.Request)) (*http.Response, error) {
	return access(ctx, url, method, header, param, body, setAuthorization)
}

func Access[T any](ctx context.Context, url string, method string, header map[string]string, param map[string]string,
	body any, setAuthorization func(*http.Request), isSuccess func(response *http.Response) bool) (T, error) {
	var ret T
	resp, err := access(ctx, url, method, header, param, body, setAuthorization)
	if err != nil {
		return ret, fmt.Errorf("access %s failed: %w", url, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ret, fmt.Errorf("read resp body failed: %w", err)
	}
	slog.InfoContext(ctx, "[Access]read resp body success")
	if isSuccess != nil && !isSuccess(resp) {
		return ret, fmt.Errorf("is success return false: %s", string(respBody))
	}
	if len(respBody) != 0 {
		err = json.Unmarshal(respBody, &ret)
		if err != nil {
			return ret, fmt.Errorf("unmarshal resp body failed: %w", err)
		}
	}
	return ret, nil
}

func access(ctx context.Context, url string, method string, header map[string]string, param map[string]string,
	body any, setAuthorization func(*http.Request)) (*http.Response, error) {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader := bytes.NewReader(bodyJSON)
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	if setAuthorization != nil {
		setAuthorization(req)
	}
	q := req.URL.Query()
	for k, v := range param {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	for k, v := range header {
		req.Header.Add(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("access %s failed: %w", url, err)
	}
	return resp, nil
}
