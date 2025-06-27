package gitmock

import (
	"math/rand/v2"
	"net/http"
	"strconv"

	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
)

type HttpMock struct {
	method      string
	path        string
	requestBody string
	body        string
	status      int
}

func SetupMocks(mocks []HttpMock) *httpmock.Mocker {
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}

	for _, mock := range mocks {
		if mock.requestBody != "" {
			fakeHTTP.RegisterResponderWithBody(
				mock.method,
				mock.path,
				mock.requestBody,
				httpmock.NewStringResponse(mock.status, mock.body),
			)
		} else {
			fakeHTTP.RegisterResponder(
				mock.method,
				mock.path,
				httpmock.NewStringResponse(mock.status, mock.body),
			)
		}
	}

	return fakeHTTP
}

func MockStackUser() HttpMock {
	return HttpMock{
		method: http.MethodGet,
		path:   "/api/v4/user",
		status: http.StatusOK,
		body:   `{ "username": "stack_guy" }`,
	}
}

func MockPostStackMR(source, target, project string) HttpMock {
	return HttpMock{
		method: http.MethodPost,
		path:   "/api/v4/projects/stack_guy%2Fstackproject/merge_requests",
		status: http.StatusOK,
		requestBody: `{
				"title": "",
				"source_branch":"` + source + `",
				"target_branch":"` + target + `",
				"assignee_id":0,
				"target_project_id": ` + project + `,
				"remove_source_branch":true
			}`,
		body: `{
			"title": "Test MR",
			"iid": ` + strconv.Itoa(rand.IntN(100)) + `,
			"source_branch":"` + source + `",
			"target_branch":"` + target + `"
		}`,
	}
}

func MockPutStackMR(target, iid, project string) HttpMock {
	return HttpMock{
		method:      http.MethodPut,
		path:        "/api/v4/projects/" + project + "/merge_requests/" + iid,
		status:      http.StatusOK,
		requestBody: `{"target_branch":"` + target + `"}`,
		body:        `{}`,
	}
}

func MockListStackMRsByBranch(branch, iid string) HttpMock {
	return HttpMock{
		method: http.MethodGet,
		path:   "/api/v4/projects/stack_guy%2Fstackproject/merge_requests?per_page=30&source_branch=" + branch,
		status: http.StatusOK,
		body:   "[" + MrMockStackData(branch, iid) + "]",
	}
}

func MockListOpenStackMRsByBranch(branch, iid string) HttpMock {
	return HttpMock{
		method: http.MethodGet,
		path:   "/api/v4/projects/stack_guy%2Fstackproject/merge_requests?per_page=30&source_branch=" + branch + "&state=opened",
		status: http.StatusOK,
		body:   "[" + MrMockStackData(branch, iid) + "]",
	}
}

func MockGetStackMR(branch, iid string) HttpMock {
	return HttpMock{
		method: http.MethodGet,
		path:   "https://gitlab.com/api/v4/projects/stack_guy%2Fstackproject/merge_requests/" + iid,
		status: http.StatusOK,
		body:   MrMockStackData(branch, iid),
	}
}

func MrMockStackData(branch, iid string) string {
	return `{
				"id": ` + iid + `,
				"iid": ` + iid + `,
				"project_id": 3,
				"title": "test mr title",
				"target_branch": "main",
				"source_branch": "` + branch + `",
				"description": "test mr description` + iid + `",
				"author": {
					"id": 1,
					"username": "admin"
				},
				"state": "opened"
			}`
}
