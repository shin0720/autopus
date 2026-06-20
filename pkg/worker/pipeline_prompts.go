package worker

import "fmt"

// Phase-specific prompt wrappers inject role context for each phase.

func (pe *PipelineExecutor) plannerPrompt(input string) string {
	return fmt.Sprintf("You are the PLANNER phase. Analyze the task and create an execution plan.\n\n%s", input)
}

func (pe *PipelineExecutor) executorPrompt(plannerOutput string) string {
	return fmt.Sprintf("You are the EXECUTOR phase. Implement the plan below.\n\n%s", plannerOutput)
}

func (pe *PipelineExecutor) testerPrompt(executorOutput string) string {
	return fmt.Sprintf("You are the TESTER phase. Write and run tests for the implementation below.\n\n%s", executorOutput)
}

func (pe *PipelineExecutor) reviewerPrompt(testerOutput string) string {
	return fmt.Sprintf("You are the REVIEWER phase. Review the implementation and test results below.\n\n%s", testerOutput)
}
