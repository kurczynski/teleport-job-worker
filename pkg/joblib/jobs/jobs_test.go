package jobs

import (
	"github.com/kurczynski/teleport-job-worker/internal/clock"
	"reflect"
	"testing"
	"time"
)

type testClock struct {
	time time.Time
}

func (tc *testClock) Now() time.Time {
	return tc.time
}

func UnixEpoch() time.Time {
	return time.UnixMilli(0).UTC()
}

// TODO: Add some more tests like this to ensure actions on a job work properly
func TestJob_updateStatus(t *testing.T) {
	type fields struct {
		statusChanges []StatusChange
		clock         clock.Clock
	}
	type args struct {
		status Status
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []StatusChange
	}{
		{
			name: "Should update initial job status",
			fields: fields{
				statusChanges: make([]StatusChange, 0),
				clock:         &testClock{time: UnixEpoch()},
			},
			args: args{status: ReadyStatus},
			want: []StatusChange{{Status: ReadyStatus, ChangedAt: UnixEpoch()}},
		},
		{
			name: "Should update existing job status",
			fields: fields{
				statusChanges: []StatusChange{
					{Status: RunningStatus, ChangedAt: UnixEpoch()},
				},
				clock: &testClock{time: UnixEpoch()},
			},
			args: args{status: StoppedStatus},
			want: []StatusChange{
				{Status: RunningStatus, ChangedAt: UnixEpoch()},
				{Status: StoppedStatus, ChangedAt: UnixEpoch()},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &Job{
				statusChanges: tt.fields.statusChanges,
				clock:         tt.fields.clock,
			}
			j.updateStatus(tt.args.status)

			if got := j.statusChanges; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("updateStatus() = %v, want %v", j.statusChanges, tt.want)
			}
		})
	}
}
