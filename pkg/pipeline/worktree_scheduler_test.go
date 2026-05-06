package pipeline_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insajin/autopus-adk/pkg/pipeline"
)

func TestWorktreeScheduler_DefaultCapQueuesOverflowByTaskID(t *testing.T) {
	t.Parallel()

	taskIDs := []string{"T7", "T3", "T1", "T6", "T2", "T5", "T4"}

	schedule := pipeline.ScheduleWorktreeTasks(taskIDs)

	assert.Equal(t, []string{"T1", "T2", "T3", "T4", "T5"}, schedule.ActiveTaskIDs)
	assert.Equal(t, []string{"T6", "T7"}, schedule.QueuedTaskIDs)
	assert.Equal(t, 5, schedule.SlotCount)
	assert.Equal(t, 5, schedule.Cap)
	assert.Equal(t, pipeline.QueueDisciplineFIFOTaskID, schedule.QueueDiscipline)
	assert.Equal(t, pipeline.ReasonWorktreeSlotCap, schedule.Reason)
	assert.Equal(t, schedule.ActiveTaskIDs, schedule.Evidence.ActiveTaskIDs)
	assert.Equal(t, schedule.QueuedTaskIDs, schedule.Evidence.QueuedTaskIDs)
}

func TestWorktreeScheduler_CustomCapUsesSameEvidenceContract(t *testing.T) {
	t.Parallel()

	schedule := pipeline.ScheduleWorktreeTasksWithCap([]string{"T10", "T2", "T1"}, 2)

	assert.Equal(t, []string{"T1", "T2"}, schedule.ActiveTaskIDs)
	assert.Equal(t, []string{"T10"}, schedule.QueuedTaskIDs)
	assert.Equal(t, 2, schedule.SlotCount)
	assert.Equal(t, 2, schedule.Cap)
	assert.Equal(t, pipeline.ReasonWorktreeSlotCap, schedule.Evidence.Reason)
}
