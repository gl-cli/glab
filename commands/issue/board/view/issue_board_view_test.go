package view

import (
	"testing"

	"github.com/stretchr/testify/assert"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func Test_issueBoardViewOptions_getListProjectIssueOptions(t *testing.T) {
	withLabelDetails := true
	labels := []string{"a", "b", "c"}
	milestone := "milestone"
	user := "user"
	state := "open"
	type fields struct {
		assignee  string
		labels    []string
		milestone string
		state     string
	}
	tests := []struct {
		name   string
		fields fields
		want   *gitlab.ListProjectIssuesOptions
	}{
		{
			name:   "return default values when passed empty options",
			fields: fields{},
			want: &gitlab.ListProjectIssuesOptions{
				WithLabelDetails: &withLabelDetails,
			},
		},
		{
			name: "return corresponding values when passing options",
			fields: fields{
				assignee:  user,
				labels:    labels,
				milestone: milestone,
				state:     state,
			},
			want: &gitlab.ListProjectIssuesOptions{
				WithLabelDetails: &withLabelDetails,
				AssigneeUsername: &user,
				Milestone:        &milestone,
				Labels:           &gitlab.LabelOptions{"a", "b", "c"},
				State:            &state,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &issueBoardViewOptions{
				assignee:  tt.fields.assignee,
				labels:    tt.fields.labels,
				milestone: tt.fields.milestone,
				state:     tt.fields.state,
			}
			got := opts.getListProjectIssueOptions()
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_issueBoardViewOptions_getListGroupIssueOptions(t *testing.T) {
	withLabelDetails := true
	labels := []string{"a", "b", "c"}
	milestone := "milestone"
	user := "user"
	state := "open"
	type fields struct {
		assignee  string
		labels    []string
		milestone string
		state     string
	}
	tests := []struct {
		name   string
		fields fields
		want   *gitlab.ListGroupIssuesOptions
	}{
		{
			name:   "return default values when passed empty options",
			fields: fields{},
			want: &gitlab.ListGroupIssuesOptions{
				WithLabelDetails: &withLabelDetails,
			},
		},
		{
			name: "return corresponding values when passing options",
			fields: fields{
				assignee:  user,
				labels:    labels,
				milestone: milestone,
				state:     state,
			},
			want: &gitlab.ListGroupIssuesOptions{
				WithLabelDetails: &withLabelDetails,
				AssigneeUsername: &user,
				Milestone:        &milestone,
				Labels:           &gitlab.LabelOptions{"a", "b", "c"},
				State:            &state,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &issueBoardViewOptions{
				assignee:  tt.fields.assignee,
				labels:    tt.fields.labels,
				milestone: tt.fields.milestone,
				state:     tt.fields.state,
			}
			got := opts.getListGroupIssueOptions()
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_buildLabelString(t *testing.T) {
	type args struct {
		labelDetails []*gitlab.LabelDetails
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "return empty string if no labeldetails are passed",
			args: args{[]*gitlab.LabelDetails{}},
			want: "",
		},
		{
			name: "return formatted string when labeldetails are passed",
			args: args{[]*gitlab.LabelDetails{
				{
					Color: "blue",
					Name:  "cold",
				},
				{
					Color: "red",
					Name:  "hot",
				},
			}},
			want: "[white:blue:-]cold[white:-:-] [white:red:-]hot[white:-:-]\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildLabelString(tt.args.labelDetails)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_mapBoardData(t *testing.T) {
	type args struct {
		projectIssueBoards      []*gitlab.IssueBoard
		projectGroupIssueBoards []*gitlab.GroupIssueBoard
	}
	type result struct {
		menuOptions  []string
		boardMetaMap map[string]boardMeta
	}
	tests := []struct {
		name string
		args args
		want result
	}{
		{
			name: "return empty map on empty inputs",
			args: args{
				projectIssueBoards:      []*gitlab.IssueBoard{},
				projectGroupIssueBoards: []*gitlab.GroupIssueBoard{},
			},
			want: result{
				menuOptions:  []string{},
				boardMetaMap: map[string]boardMeta{},
			},
		},
		{
			name: "return metadata map with input values",
			args: args{
				projectIssueBoards: []*gitlab.IssueBoard{
					{
						Name:    "projectBoard",
						Project: &gitlab.Project{Name: "project"},
						ID:      1,
					},
				},
				projectGroupIssueBoards: []*gitlab.GroupIssueBoard{
					{
						Name:  "groupBoard",
						ID:    2,
						Group: &gitlab.Group{Name: "group"},
					},
				},
			},
			want: result{
				menuOptions: []string{
					"groupBoard     (GROUP: group)",
					"projectBoard   (PROJECT: project)",
				},
				boardMetaMap: map[string]boardMeta{
					"projectBoard   (PROJECT: project)": {
						id:      1,
						name:    "projectBoard",
						project: &gitlab.Project{Name: "project"},
					},
					"groupBoard     (GROUP: group)": {
						id:    2,
						name:  "groupBoard",
						group: &gitlab.Group{Name: "group"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			menuOptions, boardMetaMap := mapBoardData(
				tt.args.projectIssueBoards,
				tt.args.projectGroupIssueBoards,
			)
			assert.Equal(t, tt.want.menuOptions, menuOptions)
			assert.Equal(t, tt.want.boardMetaMap, boardMetaMap)
		})
	}
}

func Test_filterIssues(t *testing.T) {
	type args struct {
		boardLists []*gitlab.BoardList
		issues     []*gitlab.Issue
		targetList *gitlab.BoardList
		opts       *issueBoardViewOptions
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "return empty string on no matches",
			want: "",
		},
		{
			name: "filter out closed issues when targetList is not the 'closed' list",
			args: args{
				boardLists: []*gitlab.BoardList{{Label: &gitlab.Label{Name: "A"}}},
				issues:     []*gitlab.Issue{{Labels: []string{"A"}, State: "closed"}},
				targetList: &gitlab.BoardList{Label: &gitlab.Label{Name: "A"}},
				opts:       &issueBoardViewOptions{},
			},
			want: "",
		},
		{
			name: "filter out issues not in the 'closed' state when populating the 'closed' list",
			args: args{
				boardLists: []*gitlab.BoardList{
					{Label: &gitlab.Label{Name: "Closed"}},
					{Label: &gitlab.Label{Name: "A"}},
				},
				issues:     []*gitlab.Issue{{Labels: []string{"A"}, State: "opened"}},
				targetList: &gitlab.BoardList{Label: &gitlab.Label{Name: "Closed"}},
				opts:       &issueBoardViewOptions{state: closed},
			},
			want: "",
		},
		{
			name: "filter out issues labeled for other board lists when iterating over the 'open' list",
			args: args{
				boardLists: []*gitlab.BoardList{
					{Label: &gitlab.Label{Name: "Open"}},
					{Label: &gitlab.Label{Name: "A"}},
				},
				issues:     []*gitlab.Issue{{Labels: []string{"A"}, State: "opened"}},
				targetList: &gitlab.BoardList{Label: &gitlab.Label{Name: "Open"}},
				opts:       &issueBoardViewOptions{state: opened},
			},
			want: "",
		},
		{
			name: "return formatted string on successful filter and match",
			args: args{
				boardLists: []*gitlab.BoardList{
					{Label: &gitlab.Label{Name: "A"}},
					{Label: &gitlab.Label{Name: "B"}},
					{Label: &gitlab.Label{Name: "C"}},
				},
				issues: []*gitlab.Issue{
					{
						Assignee:     &gitlab.IssueAssignee{Username: "user"},
						Labels:       []string{"A"},
						LabelDetails: []*gitlab.LabelDetails{{Name: "A", Color: "green"}},
						Title:        "Issue",
						IID:          1,
					},
				},
				targetList: &gitlab.BoardList{Label: &gitlab.Label{Name: "A"}},
				opts:       &issueBoardViewOptions{},
			},
			want: "[white::b]Issue\n[white:green:-]A[white:-:-]\n[green:-:-]#1[darkgray] - user\n\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterIssues(
				tt.args.boardLists,
				tt.args.issues,
				tt.args.targetList,
				tt.args.opts,
			)
			assert.Equal(t, tt.want, got)
		})
	}
}
