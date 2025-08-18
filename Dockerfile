# 1단계: 빌드 환경 (코드를 컴파일하는 역할)
FROM golang:1.23-alpine AS builder

# 빌드에 필요한 도구 설치 및 캐시 정리
RUN apk add --no-cache git && \
    go env -w GOCACHE=/tmp/go-cache && \
    go env -w GOMODCACHE=/tmp/go-mod

# 작업 디렉토리 설정
WORKDIR /app

# 의존성 파일만 먼저 복사하고 다운로드 (레이어 캐싱 최적화)
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# 소스 코드 복사
COPY . .

# 최적화된 빌드 (메모리 사용량 제한, 병렬 빌드 제한)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o /politisian_server . && \
    # 빌드 완료 후 캐시 정리
    go clean -cache -modcache


# 2단계: 실행 환경 (빌드된 결과물만 실행하는 역할)
# 가장 가벼운 리눅스 중 하나인 Alpine Linux를 기반으로 최종 실행 환경을 구성합니다.
FROM alpine:latest

# 작업 디렉토리 설정
WORKDIR /root/

# 빌드 환경(builder)에서 컴파일된 실행 파일만 복사합니다.
COPY --from=builder /politisian_server .

# 빌드 환경(builder)에서 프론트엔드 파일들을 복사합니다.
COPY --from=builder /app/frontend ./frontend/

# 외부에 노출할 포트를 선언합니다. (HTTP API 서버 및 CometBFT RPC)
EXPOSE 8080 26657

# 이 컨테이너가 실행될 때 최종적으로 시작될 명령어를 정의합니다.
CMD ["./politisian_server"] 