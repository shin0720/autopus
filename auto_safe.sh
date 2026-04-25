#!/bin/bash
# 메인 autopus 3인 협업 무중단 실행 보장 스크립트
export CLAUDE_CODE_NON_INTERACTIVE=true
export CLAUDE_CODE_USE_CONTINUOUS_MODE=true
export AUTO_CONFIRM=true

CMD=$1
shift

# 현재 폴더의 auto를 실행하되, 없으면 메인 경로의 auto를 실행
AUTO_BIN="./auto"
if [ ! -f "$AUTO_BIN" ]; then
    AUTO_BIN="/mnt/c/Users/SAMSUNG/autopus/auto"
fi

if [[ "$CMD" == "plan" || "$CMD" == "go" || "$CMD" == "brainstorm" || "$CMD" == "secure" || "$CMD" == "review" ]]; then
    echo "🚀 [Main] 3인 AI 협업 분석을 시작합니다... (안전 모드)"
    "$AUTO_BIN" orchestra run "$CMD" "$@"
else
    "$AUTO_BIN" "$CMD" "$@"
fi
