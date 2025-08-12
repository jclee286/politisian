#!/bin/bash
# 서버를 시작하기 전에 이전 프로세스를 종료합니다.
echo "이전 서버 프로세스 종료 시도..."
pkill -f politisian_server || true
sleep 2

# Go 소스를 빌드하여 실행 파일을 생성합니다.
# -o politisian_server는 출력 파일의 이름을 지정합니다.
echo "Go 소스 코드 빌드 중..."
go build -o politisian_server .
if [ $? -ne 0 ]; then
    echo "빌드 실패! 스크립트를 종료합니다."
    exit 1
fi
echo "빌드 성공."

# CometBFT 초기화 (필요한 경우)
# .cometbft 디렉토리가 없으면 초기화 명령을 실행합니다.
if [ ! -d ".cometbft" ]; then
    echo ".cometbft 디렉토리가 없으므로 새로 초기화합니다."
    # 여기서는 cometbft 바이너리가 시스템 PATH에 설치되어 있다고 가정합니다.
    cometbft init --home .cometbft
    if [ $? -ne 0 ]; then
        echo "CometBFT 초기화 실패! 스크립트를 종료합니다."
        exit 1
    fi
    echo "CometBFT 초기화 성공."
fi

# 애플리케이션 서버를 백그라운드에서 실행하고,
# 표준 출력(stdout)은 stdout.log 파일에, 표준 에러(stderr)는 stderr.log 파일에 저장합니다.
echo "서버 시작 중... 로그는 stdout.log와 stderr.log 파일을 확인하세요."
nohup ./politisian_server > stdout.log 2> stderr.log &
sleep 2

# 서버가 정상적으로 실행되었는지 확인
if pgrep -f "politisian_server" > /dev/null; then
    echo "서버가 성공적으로 시작되었습니다."
else
    echo "서버 시작에 실패했습니다. stderr.log 파일을 확인해주세요."
fi 