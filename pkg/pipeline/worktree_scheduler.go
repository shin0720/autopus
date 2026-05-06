package pipeline

import (
	"sort"
	"strconv"
	"strings"
)

const (
	// @AX:NOTE: [AUTO] slot cap and FIFO discipline strings define deterministic worktree scheduling evidence.
	// DefaultWorktreeSlotCap is the default maximum concurrent executor worktrees.
	DefaultWorktreeSlotCap = maxWorktrees
	// DefaultMaxWorktreeSlots is an alias for callers using slot terminology.
	DefaultMaxWorktreeSlots = DefaultWorktreeSlotCap
	// QueueDisciplineFIFOTaskID identifies deterministic task-id FIFO scheduling.
	QueueDisciplineFIFOTaskID = "fifo_task_id"
)

// @AX:ANCHOR: [AUTO] worktree scheduling evidence schema consumed by parallel runner safety events.
// @AX:REASON: Active/queued task lists, cap, and queue discipline document bounded parallelism for downstream diagnostics.
// WorktreeSchedule is the worktree slot decision for a task set.
type WorktreeSchedule struct {
	ActiveTaskIDs   []string         `json:"active_task_ids"`
	QueuedTaskIDs   []string         `json:"queued_task_ids"`
	SlotCount       int              `json:"slot_count"`
	Cap             int              `json:"cap"`
	QueueDiscipline string           `json:"queue_discipline"`
	Reason          SafetyReasonCode `json:"reason"`
	Evidence        DegradedEvidence `json:"evidence"`
}

// ScheduleWorktreeTasks schedules task IDs with the default worktree slot cap.
func ScheduleWorktreeTasks(taskIDs []string) WorktreeSchedule {
	return ScheduleWorktreeTasksWithCap(taskIDs, DefaultWorktreeSlotCap)
}

// ScheduleWorktreeTasksWithCap schedules task IDs into active and queued slots.
func ScheduleWorktreeTasksWithCap(taskIDs []string, cap int) WorktreeSchedule {
	if cap <= 0 {
		cap = DefaultWorktreeSlotCap
	}

	ordered := orderedTaskIDs(taskIDs)
	slotCount := len(ordered)
	if slotCount > cap {
		slotCount = cap
	}

	active := cloneStrings(ordered[:slotCount])
	queued := cloneStrings(ordered[slotCount:])
	evidence := DegradedEvidence{
		Reason:          ReasonWorktreeSlotCap,
		ActiveTaskIDs:   active,
		QueuedTaskIDs:   queued,
		SlotCount:       slotCount,
		Cap:             cap,
		QueueDiscipline: QueueDisciplineFIFOTaskID,
	}

	return WorktreeSchedule{
		ActiveTaskIDs:   active,
		QueuedTaskIDs:   queued,
		SlotCount:       slotCount,
		Cap:             cap,
		QueueDiscipline: QueueDisciplineFIFOTaskID,
		Reason:          ReasonWorktreeSlotCap,
		Evidence:        evidence,
	}
}

func orderedTaskIDs(taskIDs []string) []string {
	ordered := cloneStrings(taskIDs)
	sort.SliceStable(ordered, func(i, j int) bool {
		return taskIDLess(ordered[i], ordered[j])
	})
	return ordered
}

func taskIDLess(left, right string) bool {
	leftPrefix, leftNum, leftOK := splitTaskID(left)
	rightPrefix, rightNum, rightOK := splitTaskID(right)
	if leftOK && rightOK && leftPrefix == rightPrefix && leftNum != rightNum {
		return leftNum < rightNum
	}
	return left < right
}

func splitTaskID(taskID string) (string, int, bool) {
	idx := len(taskID)
	for idx > 0 && taskID[idx-1] >= '0' && taskID[idx-1] <= '9' {
		idx--
	}
	if idx == len(taskID) {
		return taskID, 0, false
	}
	n, err := strconv.Atoi(taskID[idx:])
	if err != nil {
		return taskID, 0, false
	}
	return strings.ToUpper(taskID[:idx]), n, true
}
