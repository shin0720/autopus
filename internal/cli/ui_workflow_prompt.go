package cli

import (
	"fmt"
	"regexp"
	"strings"
)

// codeProducingAgents are agents whose successful output MUST include real file
// modifications. If they end with a question to the user or report "## 수정된 파일\n- 없음",
// autopus rejects the response and re-runs the agent with a stricter prompt.
var codeProducingAgents = map[string]bool{
	"exec": true,
	"deep": true,
	"dbug": true,
	"fend": true,
	"test": true,
	"val":  true,
	"sec":  true,
	"perf": true,
}

// uiProducingAgents are agents that touch UI surfaces. Used to determine when
// the Planner / Spec Writer must produce UI screen specs (Fix C).
var uiProducingAgents = map[string]bool{
	"fend": true,
	"uxv":  true,
}

// buildAgentPrompt constructs the full prompt for an agent run, including:
//   - the role + user instruction + code context (existing behavior)
//   - generic action principles and response rules (existing behavior)
//   - the ## 상류 부족 보고 self-escalation section (Fix B)
//   - agent-specific augmentation for Planner / Spec Writer (Fix C)
func buildAgentPrompt(agentID, agentName, userPrompt, codeContext string) string {
	roleSpecific := agentSpecificDirective(agentID)

	body := fmt.Sprintf(
		"역할: %s\n\n지시: %s\n\n코드 컨텍스트:\n%s\n\n",
		agentName, userPrompt, codeContext,
	)

	if roleSpecific != "" {
		body += "---\n[역할별 추가 지시]\n" + roleSpecific + "\n\n"
	}

	body += "---\n[행동 원칙]\n" +
		"당신은 분석가가 아니라 실행자입니다. 지시가 오면 즉시 파일을 수정하거나 생성하세요.\n" +
		"사용자에게 질문하거나 승인을 요청하지 마세요. 모든 결정은 스스로 내리고 이유를 문서에 기록하세요.\n\n" +
		"절대 하지 말아야 할 것:\n" +
		"- 응답 마지막을 '어떤 작업이 필요하신가요?', '무엇을 도와드릴까요?', '다음 단계를 선택하세요' 등 질문이나 선택지 목록으로 끝내는 것\n" +
		"- 결정이 필요한 상황에서 사용자에게 묻는 것 — 스스로 최선의 선택을 하고 이유를 기록하세요\n" +
		"- 이미 확정된 사항을 다시 질문하거나 '승인하시겠습니까?' 같은 확인 요청\n\n" +
		"[상류 작업 부족 처리]\n" +
		"이전 작업자가 일을 안 했거나 사전 명세가 비어있어서 본인 역할을 수행할 수 없을 때만, " +
		"코드 작업을 시도하지 말고 응답 끝에 정확히 다음 형식으로 보고하세요:\n\n" +
		"## 상류 부족 보고\n" +
		"필요한 사전 작업자: <agentId>\n" +
		"부족한 항목: (구체적으로 무엇이 빠졌고 본인이 왜 못하는지)\n\n" +
		"⚠️ <agentId> 는 반드시 프롬프트 안에 별도로 제시되는 [연결된 상류 작업자] 목록에서 골라야 합니다. " +
		"그 목록에 없는 ID 를 적으면 회귀가 거부됩니다 (사용자가 그래프상 그 연결선을 안 그렸다는 뜻). " +
		"목록이 비어있으면 회귀할 수 없으니, 본인이 가진 정보로 최대한 결정하고 작업하거나 사유만 적어 종료하세요.\n\n" +
		"실제로 본인 역할을 수행할 수 있는 정보가 충분하면 이 섹션을 절대 출력하지 마세요. " +
		"임의로 자기 영역 밖의 일을 떠맡지 마세요.\n\n" +
		"[응답 규칙]\n" +
		"1. 반드시 한국어로 응답하세요.\n" +
		"2. 결정이 필요한 항목이 있으면 스스로 최선의 선택을 하고 즉시 실행하세요:\n" +
		"   - Edit 도구로 파일을 직접 열어 수정하세요.\n" +
		"   - 선택한 이유를 ## 결정 사항 섹션에 간단히 기록하세요.\n\n" +
		"3. 작업 완료 후 반드시 아래 형식으로 마무리하고 즉시 종료하세요:\n\n" +
		"## 작업 요약\n(수행한 작업 1~2줄)\n\n" +
		"## 결정 사항\n(스스로 내린 결정과 이유 — 결정이 없으면 이 섹션 생략)\n\n" +
		"## 수정된 파일\n- 파일경로: 무엇을 어떻게 수정했는지\n\n" +
		"## 추가된 파일\n- 파일경로: 어떤 목적으로 생성했는지\n\n" +
		"## 주요 변경 내용\n(각 변경사항 구체적 설명)\n\n" +
		"## 작업 후 검증\n" +
		"(사용자가 변경사항이 제대로 적용됐는지 직접 확인할 수 있는 절차를 단계별로 적으세요. " +
		"비기술자도 따라할 수 있게 풀어쓰세요. 다음 4종을 가능한 모두 포함:\n" +
		"  1. 실행 명령 — 예: `cd backend && uvicorn app.main:app --reload`\n" +
		"  2. 확인 URL/화면 — 예: `http://localhost:3000/results` 에서 결과 카드 클릭\n" +
		"  3. 기대 결과 — 예: 카드에 UUID 대신 한글 이름과 한자가 표시됨\n" +
		"  4. 자동 검증 명령 — 예: `cd frontend && npm run lint && npx tsc --noEmit`)\n\n" +
		"4. 기술 용어 첫 등장 시 괄호로 설명 추가.\n" +
		"예시: Railway(서버를 인터넷에 올려주는 호스팅 서비스), PG(카드 결제를 처리해주는 회사)"

	return body
}

// agentSpecificDirective returns extra instructions for specific agent roles.
// Fix C: Planner and Spec Writer MUST produce UI screen specs when the project
// involves a user-facing interface, so downstream Frontend has concrete work.
func agentSpecificDirective(agentID string) string {
	switch agentID {
	case "planner":
		return strings.TrimSpace(`
프로젝트에 사용자 인터페이스(웹/모바일/데스크톱 화면) 가 포함되는지 판단하세요. 포함되면:
- 화면 목록을 빠짐없이 식별합니다 (예: 랜딩, 결과 상세, 마이페이지, 결제 등).
- 각 화면마다 다음을 명시한 UI 화면 명세를 산출물에 포함하세요:
  · 화면 목적 한 줄
  · 표시할 핵심 정보 (필드 단위)
  · 사용자가 할 수 있는 액션 (버튼, 링크)
  · 비회원/회원 같은 상태별 동작
  · 빈 상태/에러 상태 처리
- 디자인 톤(미니멀/캐주얼/한방 등) 과 기준 컬러를 한 문장으로 결정합니다.
이 항목들이 산출물에 없으면 후속 Spec Writer/Frontend 가 일을 못 합니다. 절대 생략하지 마세요.`)
	case "spec":
		return strings.TrimSpace(`
이전 Planner 의 UI 화면 목록을 SPEC 으로 확장하세요:
- 화면당 별도 SPEC (또는 한 SPEC 의 별도 acceptance 항목) 으로 분해
- 각 화면의 데이터 흐름, 상태 전이, 컴포넌트 단위 분해까지 적습니다
- 화면 명세가 없는 SPEC 은 Frontend 가 구현 불가. 누락 시 ## 상류 부족 보고 로 planner 에게 반려하세요.`)
	case "fend":
		return strings.TrimSpace(`
Spec Writer / Planner 의 UI 명세가 충분한지 먼저 확인하세요:
- 화면 목록이 있는가
- 화면당 필드/액션/상태가 명시되었는가
- 디자인 톤과 기준 컬러가 결정되었는가
하나라도 빠지면 임의로 디자인하지 말고 ## 상류 부족 보고 로 spec 또는 planner 에게 반려하세요.
충분하면 명세를 그대로 따라 구현하세요. 임의 추가/장식 금지.`)
	case "devops":
		return strings.TrimSpace(`
당신의 영역은 CI/CD, Docker, GitHub Actions, 배포 자동화 입니다.
UI 컴포넌트, 페이지, API 핸들러 같은 애플리케이션 코드는 절대 손대지 마세요.
다음 중 하나라도 빠져있으면 ## 상류 부족 보고 로 거꾸로 반려하세요:
- frontend/ 또는 backend/ 의 핵심 페이지/엔드포인트가 비어있다 → fend 또는 exec
- 빌드 가능한 상태가 아니다 → exec
- 배포 대상이 명세상 정의되지 않았다 → arch 또는 planner`)
	}
	return ""
}

// 패턴: 응답 끝부분이 사용자 질문/선택지로 끝나는지 검출
var (
	endsWithQuestionRe = regexp.MustCompile(`(?ms)(어떤\s*[작업부분]|어느\s*것|무엇을|어떻게\s*진행|어떤\s*작업을|진행할까요|확인해\s*드릴까요|선택해\s*주세요|무엇이\s*필요|어떤\s*기능)\s*\??\s*$`)
	choiceListAtEndRe  = regexp.MustCompile(`(?ms)(?:^[-*\d.)\s]+.+\n){2,}\s*$`)
	noModifiedSectionRe = regexp.MustCompile(`(?m)^##\s*수정된\s*파일\s*\n(?:\s*-\s*없음|\s*없음|\s*$|\s*\n##)`)
	hasUpstreamReportRe = regexp.MustCompile(`(?m)^##\s*상류\s*부족\s*보고\s*\n`)
)

// validateAgentOutput inspects an agent's response and decides whether it
// represents real work. Returns ("", true) when the output is acceptable, or
// (reason, false) when the agent should be re-run with a stricter prompt.
//
// Rules:
//   - If the output contains "## 상류 부족 보고", it is ACCEPTED as-is (this is
//     a valid upstream-escalation; the dashboard handles re-routing).
//   - For code-producing agents only: reject when the output has no real file
//     modifications, or ends with a user-facing question / choice list.
//   - Other agents (planner/spec/arch/expl/etc.) are not required to modify
//     files; only the question-ending check applies.
func validateAgentOutput(agentID, output string) (string, bool) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "빈 응답입니다.", false
	}
	if hasUpstreamReportRe.MatchString(trimmed) {
		// Valid upstream escalation — dashboard will reroute. Don't reject.
		return "", true
	}

	lower := strings.ToLower(trimmed)
	// Question-ending detection (applies to ALL agents).
	tail := trimmed
	if len(tail) > 600 {
		tail = tail[len(tail)-600:]
	}
	if endsWithQuestionRe.MatchString(tail) {
		return "응답이 사용자에게 되묻는 형태로 끝났습니다. 본인이 결정하고 실행하세요.", false
	}
	if strings.HasSuffix(strings.TrimRight(tail, " \t\n"), "?") && !strings.Contains(lower, "## 작업 요약") {
		return "응답이 물음표로 끝났는데 작업 요약이 없습니다.", false
	}

	if !codeProducingAgents[agentID] {
		return "", true
	}

	// Code-producing agents must show real file modifications.
	if !strings.Contains(output, "## 수정된 파일") && !strings.Contains(output, "## 추가된 파일") {
		return "코드 작업자인데 ## 수정된 파일 / ## 추가된 파일 섹션이 없습니다.", false
	}
	if noModifiedSectionRe.MatchString(output) && !strings.Contains(output, "## 추가된 파일\n-") {
		return "수정/추가된 파일이 보고되지 않았습니다. 실제로 파일을 작성하세요.", false
	}
	return "", true
}

// buildRetryPrompt prepends a strong instruction explaining why the previous
// run was rejected and what must change. Used when validateAgentOutput rejects
// an agent's output.
func buildRetryPrompt(originalPrompt, rejectionReason string, attempt int) string {
	return fmt.Sprintf(
		"🚨 자동 파이프라인 검증 실패 (시도 %d회). 이전 응답이 거부되었습니다.\n"+
			"거부 사유: %s\n\n"+
			"이번에는 다음 규칙을 반드시 지키세요:\n"+
			"- 사용자에게 어떤 형태로든 질문하거나 선택지를 제시하지 마세요.\n"+
			"- 본인의 역할 범위에서 즉시 파일을 수정/생성하세요.\n"+
			"- 만약 정말로 사전 작업이 부족하면 임의로 떠맡지 말고 응답 끝에 ## 상류 부족 보고 섹션을 출력하세요.\n"+
			"- 응답은 ## 작업 요약 / ## 수정된 파일 / ## 추가된 파일 / ## 주요 변경 내용 / ## 작업 후 검증 형식으로 종료합니다.\n\n"+
			"---\n%s",
		attempt, rejectionReason, originalPrompt,
	)
}
