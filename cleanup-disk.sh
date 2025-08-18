#!/bin/bash

echo "=== 디스크 공간 정리 시작 ==="

# 현재 디스크 사용량 확인
echo "현재 디스크 사용량:"
df -h

echo ""
echo "=== Docker 정리 시작 ==="

# Docker 컨테이너, 이미지, 볼륨, 네트워크 정리
echo "1. 중지된 컨테이너 제거..."
docker container prune -f

echo "2. 사용하지 않는 이미지 제거..."
docker image prune -a -f

echo "3. 사용하지 않는 볼륨 제거..."
docker volume prune -f

echo "4. 사용하지 않는 네트워크 제거..."
docker network prune -f

echo "5. 빌드 캐시 정리..."
docker builder prune -a -f

echo ""
echo "=== 시스템 임시 파일 정리 ==="

# 시스템 임시 파일 정리
echo "6. /tmp 디렉토리 정리..."
find /tmp -type f -atime +7 -delete 2>/dev/null || true

echo "7. 로그 파일 정리..."
find /var/log -name "*.log" -type f -size +100M -delete 2>/dev/null || true

echo "8. APT 캐시 정리..."
apt-get clean 2>/dev/null || true

echo "9. 사용하지 않는 패키지 제거..."
apt-get autoremove -y 2>/dev/null || true

echo ""
echo "=== Go 캐시 정리 ==="
echo "10. Go 모듈 캐시 정리..."
go clean -modcache 2>/dev/null || true
go clean -cache 2>/dev/null || true

echo ""
echo "=== 정리 완료 후 디스크 사용량 ==="
df -h

echo ""
echo "=== 정리 완료 ==="