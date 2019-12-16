package main

import (
	"testing"
	"time"
)

func Test_build_buildType(t *testing.T) {
	dereferenceInt := func(i int64) *int64 {
		return &i
	}

	type fields struct {
		Tag                     string
		Slug                    string
		Branch                  string
		Status                  int
		CommitHash              string
		StatusText              string
		TriggeredAt             time.Time
		BuildNumber             int64
		PullRequestID           *int64
		TriggeredWorkflow       string
		PullRequestTargetBranch string
	}
	tests := []struct {
		name   string
		fields fields
		want   buildType
	}{
		{
			name: "pull request ID nil, target branch exists",
			fields: fields{
				Tag:                     "",
				PullRequestID:           nil,
				PullRequestTargetBranch: "master",
				CommitHash:              "",
			},
			want: buildTypePullRequest,
		},
		{
			name: "pull request ID not nil, target branch exists",
			fields: fields{
				Tag:                     "",
				PullRequestID:           dereferenceInt(1),
				PullRequestTargetBranch: "master",
				CommitHash:              "",
			},
			want: buildTypePullRequest,
		},
		{
			name: "tag specified",
			fields: fields{
				Tag:                     "tag1",
				PullRequestID:           nil,
				PullRequestTargetBranch: "master",
				CommitHash:              "234abc",
			},
			want: buildTypeTag,
		},
		{
			name: "commit hash specified only",
			fields: fields{
				Tag:                     "",
				PullRequestID:           nil,
				PullRequestTargetBranch: "",
				CommitHash:              "234abc",
			},
			want: buildTypePush,
		},
		{
			name: "commit hash specified only",
			fields: fields{
				PullRequestID:           nil,
				PullRequestTargetBranch: "",
				Tag:                     "",
				CommitHash:              "",
			},
			want: buildTypeManual,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			build := build{
				Tag:                     tt.fields.Tag,
				Slug:                    tt.fields.Slug,
				Branch:                  tt.fields.Branch,
				Status:                  tt.fields.Status,
				CommitHash:              tt.fields.CommitHash,
				StatusText:              tt.fields.StatusText,
				TriggeredAt:             tt.fields.TriggeredAt,
				BuildNumber:             tt.fields.BuildNumber,
				PullRequestID:           tt.fields.PullRequestID,
				TriggeredWorkflow:       tt.fields.TriggeredWorkflow,
				PullRequestTargetBranch: tt.fields.PullRequestTargetBranch,
			}
			if got := build.buildType(); got != tt.want {
				t.Errorf("build.buildType() = %v, want %v", got, tt.want)
			}
		})
	}
}
