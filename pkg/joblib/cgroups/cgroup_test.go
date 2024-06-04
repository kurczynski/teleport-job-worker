package cgroups

import (
	"testing"
)

// TODO: Add more unit tests

func TestCgroup_withJobPath(t *testing.T) {
	type fields struct {
		jobID      string
		root       string
		workerName string
	}
	type args struct {
		resource []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "Should create job path with single resource",
			fields: fields{
				jobID:      "some-job-id",
				root:       "/cgroup/root/path",
				workerName: "some-worker",
			},
			args: args{resource: []string{"some-resource"}},
			want: "/cgroup/root/path/some-worker/some-job-id/some-resource",
		},
		{
			name: "Should create job path with multiple resources",
			fields: fields{
				jobID:      "some-job-id",
				root:       "/cgroup",
				workerName: "some-worker",
			},
			args: args{resource: []string{"resource-one", "resource-two", "resource-three"}},
			want: "/cgroup/some-worker/some-job-id/resource-one/resource-two/resource-three",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cgroup{
				jobID:      tt.fields.jobID,
				root:       tt.fields.root,
				workerName: tt.fields.workerName,
			}
			if got := c.withJobPath(tt.args.resource...); got != tt.want {
				t.Errorf("withJobPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCgroup_withWorkerPath(t *testing.T) {
	type fields struct {
		root       string
		workerName string
	}
	type args struct {
		resource []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "Should create worker path with single resource",
			fields: fields{
				root:       "/cgroup/root/path",
				workerName: "some-worker",
			},
			args: args{resource: []string{"some-resource"}},
			want: "/cgroup/root/path/some-worker/some-resource",
		},
		{
			name: "Should create worker path with multiple resources",
			fields: fields{
				root:       "/cgroup",
				workerName: "some-worker",
			},
			args: args{resource: []string{"resource-one", "resource-two", "resource-three"}},
			want: "/cgroup/some-worker/resource-one/resource-two/resource-three",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cgroup{
				root:       tt.fields.root,
				workerName: tt.fields.workerName,
			}
			if got := c.withWorkerPath(tt.args.resource...); got != tt.want {
				t.Errorf("withWorkerPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
