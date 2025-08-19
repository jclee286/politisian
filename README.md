# 정치인 공화국 (Politician Republic)

> **익명성 보장된 정치인 코인 거래소**  
> 전통적 이메일/비밀번호 로그인 + 하이브리드 블록체인 지갑 시스템

## 🏛️ 프로젝트 개요

정치인 공화국은 개발자의 익명성을 완벽히 보호하면서, 사용자들이 정치인 코인을 실제 돈(USDT/USDC)으로 거래할 수 있는 탈중앙화 거래소입니다.

### 핵심 특징
- **🔒 완전 익명 운영**: OAuth 제거로 개발자 정보 완전 차단
- **💰 실제 자산 거래**: Polygon 기반 USDT/USDC로 정치인 코인 매매
- **⚡ 하이브리드 구조**: 정치인코인(자체/무료) + 스테이블코인(Polygon)
- **🏪 오더북 거래**: 실시간 매수/매도 주문서 기반 거래
- **💎 에스크로 시스템**: 안전한 거래 보장

## 🏗️ 시스템 아키텍처

```
┌─────────────────────────────────────────┐
│              통합 지갑 (UI)              │
├─────────────────────────────────────────┤
│ 정치인 코인 (자체 블록체인 - 무료, 빠름)  │
│ - 문재인 코인: 1,000개                  │  
│ - 윤석열 코인: 500개                    │
├─────────────────────────────────────────┤
│ 스테이블 코인 (Polygon - 실제 자산)      │
│ - USDT: $1,000                         │
│ - USDC: $500                           │
│ - MATIC: 10개 (수수료용)                │
└─────────────────────────────────────────┘
```

### 기술 스택
- **Backend**: Go + CometBFT (자체 블록체인)
- **Frontend**: Vanilla HTML/CSS/JavaScript
- **External**: Polygon Network (USDT/USDC)
- **API**: Etherscan API V2 (트랜잭션 검증)
- **Deployment**: Docker + GitHub Actions CI/CD

## 📊 거래 시스템

### 오더북 거래소
```
매수 주문                      매도 주문
가격    수량                   가격    수량
1,100   500개 ←─ 매수 1호가     1,150   300개 ← 매도 1호가
1,090   800개                  1,160   700개
1,080   1,200개                1,170   500개
```

### 에스크로 안전장치
- **매수 주문**: USDT/USDC 동결 → 체결 시 정치인코인 수령
- **매도 주문**: 정치인코인 동결 → 체결 시 USDT/USDC 수령
- **실패 시**: 자동 동결 해제

## 💳 지갑 시스템

### Polygon 스테이블코인 지갑
- **주소 형태**: `0x1234...abcd` (Ethereum 호환)
- **지원 토큰**: USDT, USDC, MATIC
- **입금 방법**: 바이낸스 → Polygon 네트워크 출금
- **컨트랙트 주소**:
  - USDT: `0xc2132D05D31c914a87C6611C10748AEb04B58e8F`
  - USDC: `0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174`

### 자체 정치인코인 지갑
- **무료 거래**: 수수료 없는 즉시 전송
- **빠른 처리**: 블록 생성 5초
- **안전 보관**: CometBFT 합의 알고리즘

## 🚀 배포 및 실행

### 로컬 개발
```bash
# 의존성 설치
go mod tidy

# 애플리케이션 실행
go run main.go

# 웹 브라우저에서 접속
http://localhost:8080
```

### Docker 실행
```bash
# 컨테이너 빌드 및 실행
docker-compose up --build

# 백그라운드 실행
docker-compose up -d
```

### 프로덕션 배포
```bash
# GitHub에 푸시하면 자동 배포
git push origin main
# → GitHub Actions가 서버에 자동 배포
```

## 📁 프로젝트 구조

```
politisian/
├── 🚀 main.go                    # 애플리케이션 진입점
├── 📁 app/                       # 블록체인 애플리케이션
│   ├── abci.go                   # ABCI 트랜잭션 처리
│   ├── app.go                    # 애플리케이션 초기화
│   └── state.go                  # 상태 관리
├── 📁 server/                    # HTTP API 서버
│   ├── auth.go                   # 전통 인증 (이메일/패스워드)
│   ├── handlers.go               # 기본 API 핸들러
│   ├── polygon_handlers.go       # USDT/USDC 입출금 API
│   ├── polygon_wallet.go         # Polygon 지갑 생성/검증
│   ├── trade_handlers.go         # 거래소 API
│   └── server.go                 # 서버 라우팅
├── 📁 frontend/                  # 웹 인터페이스
│   ├── index.html               # 대시보드/거래소
│   ├── login.html               # 로그인
│   └── signup.html              # 회원가입
├── 📁 pkg/types/                 # 데이터 구조
│   └── types.go                 # 공통 타입 정의
├── 🐳 Dockerfile                 # 컨테이너 이미지
├── 🔧 docker-compose.yml         # 개발환경 설정
└── 📚 README.md                  # 이 문서
```

## 🔐 보안 및 익명성

### 개발자 익명성 보장
- **OAuth 완전 제거**: 구글 로그인 등 제3자 인증 차단
- **전통 인증**: 이메일/패스워드만 사용
- **개인정보 최소화**: 이메일 인증 생략

### 사용자 보안
- **PIN 기반 지갑**: 이중 보안 (로그인 + PIN)
- **bcrypt 해싱**: 패스워드 암호화 저장
- **에스크로 보호**: 거래 실패 시 자동 환불
- **세션 관리**: 메모리 기반 세션 스토어

## 💰 경제 모델

### 정치인 코인
- **발행량**: 정치인당 1,000만개 고정
- **초기 지급**: 신규 가입 시 선택한 3명당 100개씩
- **거래**: USDT/USDC로 매매 가능
- **가격**: 오더북 기반 시장 결정

### 수수료 구조
- **정치인코인 거래**: 무료 (자체 블록체인)
- **USDT/USDC 입금**: 바이낸스 출금 수수료만 (약 1 USDT)
- **USDT/USDC 출금**: Polygon 네트워크 수수료 (약 0.1 USDT)

## 🔗 외부 연동

### Polygon Network
- **RPC**: `https://polygon-rpc.com`
- **트랜잭션 검증**: Etherscan API V2
- **지원 지갑**: MetaMask, Trust Wallet 등

### API 키 설정
```go
// 환경변수 설정
export ETHERSCAN_API_KEY="your_api_key_here"

// 또는 코드에서 직접 설정
func getPolygonAPIKey() string {
    return "RTKWX1EIEXG3V59WFU9MKTNHQIRKKCNS2U"
}
```

## 🎯 사용자 시나리오

### 신규 사용자
1. **회원가입**: 이메일/패스워드 + 프로필 입력
2. **정치인 선택**: 3명 선택 시 300개 코인 지급
3. **지갑 생성**: Polygon 주소 자동 생성
4. **USDT 입금**: 바이낸스 → Polygon → 우리 주소

### 거래
1. **매수 주문**: USDT로 정치인코인 구매 주문
2. **에스크로**: USDT 자동 동결
3. **매칭**: 매도 주문과 가격 매칭 시 체결
4. **정산**: 정치인코인 받고 USDT 차감

### 출금
1. **출금 신청**: USDT/USDC 출금 주소 입력
2. **PIN 인증**: 지갑 PIN 입력
3. **Polygon 전송**: 실제 블록체인으로 토큰 전송
4. **확인**: 1-3분 내 MetaMask 등에서 수령 확인

## 📈 개발 로드맵

### ✅ 완료된 기능
- 전통적 이메일/패스워드 인증 시스템
- Polygon 기반 USDT/USDC 지갑
- 오더북 거래소 시스템
- 에스크로 안전장치
- 실제 API 키 연동
- Docker 배포 자동화

### 🚧 진행 중
- UI/UX 개선 및 통합 지갫 표시
- 실시간 가격 차트
- 모바일 최적화

### 📋 향후 계획
- 다양한 스테이블코인 지원 (BUSD, DAI 등)
- 레버리지 거래 기능
- 정치인 코인 스테이킹 보상
- 소셜 기능 (팔로우, 커뮤니티)

## 🤝 기여 가이드

### 개발 환경 설정
```bash
# 저장소 클론
git clone https://github.com/jclee286/politisian.git
cd politisian

# Go 모듈 설치
go mod tidy

# 로컬 실행
go run main.go
```

### 커밋 컨벤션
- `feat:` 새로운 기능 추가
- `fix:` 버그 수정
- `refactor:` 코드 리팩토링
- `docs:` 문서 수정

## 📞 연락처

- **GitHub**: https://github.com/jclee286/politisian
- **Issues**: https://github.com/jclee286/politisian/issues

---

**⚠️ 주의사항**: 이 프로젝트는 교육 및 연구 목적으로 개발되었습니다. 실제 투자 시 모든 책임은 사용자에게 있습니다.

**🛡️ 면책조항**: 개발자는 익명성을 보장하며, 사용자의 거래 손실에 대해 책임지지 않습니다.