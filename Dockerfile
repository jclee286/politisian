# 1단계: 빌드 환경 (코드를 컴파일하는 역할)
# Go 1.23 버전의 가벼운 Alpine Linux를 기반으로 빌드 서버를 구성합니다.
FROM golang:1.23-alpine AS builder

# 작업 디렉토리 설정
WORKDIR /app

# 의존성 관리를 위해 go.mod와 go.sum 파일을 먼저 복사합니다.
# 이렇게 하면, 소스 코드가 변경되지 않는 한 불필요한 의존성 다운로드를 건너뛰어 빌드 속도를 높일 수 있습니다.
COPY go.mod go.sum ./
RUN go mod download

# 나머지 모든 소스 코드를 복사합니다.
COPY . .

# CGO_ENABLED=0는 다른 시스템 라이브러리에 대한 의존성이 없는 정적 바이너리를 만듭니다.
# 이렇게 하면 어떤 리눅스 환경에서도 실행 파일이 독립적으로 실행될 수 있습니다.
# -o /politisian_server는 컴파일 결과물이 / 경로에 politisian_server라는 이름으로 저장되도록 합니다.
RUN CGO_ENABLED=0 go build -o /politisian_server .


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