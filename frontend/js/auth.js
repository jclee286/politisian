// 인증 관련 함수들

// 사용자 프로필 로드
function loadUserProfile() {
    console.log('👤 사용자 프로필 로드 시작');
    
    fetch('/api/user/profile')
        .then(response => {
            console.log('📡 프로필 API 응답:', response.status);
            
            if (response.status === 401) {
                console.log('🔒 401 인증 오류 - 세션 정보로 대체');
                loadSessionInfo();
                return null;
            }
            
            if (!response.ok) {
                console.log('❌ 프로필 API 실패, 세션 정보로 대체');
                loadSessionInfo();
                return null;
            }
            
            return response.json();
        })
        .then(data => {
            if (data) {
                console.log('✅ 프로필 데이터 로드 성공:', data);
                
                // 전역 변수에 프로필 데이터 저장
                currentUserProfileData = data;
                
                setWalletAddress(data.wallet || data.walletAddress || '지갑 주소 없음');
                
                // 백업 데이터 저장
                saveBackupData(data.wallet || data.walletAddress, {
                    email: data.email,
                    address: data.address
                });
                
                updateDashboardUI(data);
                loadProposals();
                loadRegisteredPoliticians();
                loadTradingData();
            }
        })
        .catch(error => {
            console.error('❌ 프로필 로드 실패:', error);
            loadSessionInfo();
        });
}

// 세션 정보로 대체 로드
function loadSessionInfo() {
    console.log('🔑 세션 정보로 대체 로드 시도');
    
    fetch('/api/user/session-info')
        .then(response => {
            console.log('📡 세션 API 응답 상태:', response.status);
            if (response.ok) {
                return response.json();
            }
            throw new Error(`세션 정보 로드 실패: ${response.status}`);
        })
        .then(sessionData => {
            console.log('✅ 세션 데이터 로드 성공:', sessionData);
            
            setWalletAddress(sessionData.walletAddress || '지갑 주소 없음');
            
            saveBackupData(sessionData.walletAddress, {
                name: sessionData.name,
                email: sessionData.email,
                userId: sessionData.userId
            });
            
            if (politicianCoinsListElem) {
                politicianCoinsListElem.innerHTML = '<li>프로필 정보를 완전히 불러오지 못했습니다.</li>';
            }
            
            setTimeout(() => {
                loadProposals();
                loadRegisteredPoliticians();
            }, 2000);
        })
        .catch(error => {
            console.error('❌ 세션 정보도 로드 실패:', error);
            handleCompleteFailure();
        });
}

// 완전한 로드 실패 시 복구 전략
function handleCompleteFailure() {
    console.log('🚨 완전한 데이터 로드 실패 - 복구 시도');
    
    const backupWallet = localStorage.getItem('backup_wallet_address');
    const backupUserInfo = localStorage.getItem('backup_user_info');
    
    if (backupWallet) {
        console.log('💾 로컬 백업에서 지갑 주소 복구');
        setWalletAddress(backupWallet);
        
        if (backupUserInfo) {
            try {
                const userInfo = JSON.parse(backupUserInfo);
                console.log('백업 사용자 정보 복구:', userInfo);
            } catch (e) {
                console.log('백업 사용자 정보 파싱 실패');
            }
        }
        
        showRecoveryMessage();
        return;
    }
    
    setWalletAddress('세션이 만료되었습니다');
    if (politicianCoinsListElem) {
        politicianCoinsListElem.innerHTML = '<li style="color: #e74c3c;">로그인이 필요합니다</li>';
    }
    
    showReloginPrompt();
}

// 복구 메시지 표시
function showRecoveryMessage() {
    const recoveryMsg = document.createElement('div');
    recoveryMsg.style.cssText = `
        position: fixed; top: 20px; right: 20px; 
        background: #f39c12; color: white; 
        padding: 15px 20px; border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.3);
        z-index: 1000; font-size: 14px;
    `;
    recoveryMsg.innerHTML = `
        ⚠️ 백업 데이터에서 복구됨<br>
        <small>새로고침 후 다시 로그인을 권장합니다</small>
    `;
    document.body.appendChild(recoveryMsg);
    
    setTimeout(() => recoveryMsg.remove(), 10000);
}

// 재로그인 프롬프트 표시
function showReloginPrompt() {
    const reloginDiv = document.createElement('div');
    reloginDiv.style.cssText = `
        position: fixed; top: 50%; left: 50%; 
        transform: translate(-50%, -50%);
        background: white; padding: 30px; 
        border-radius: 12px; box-shadow: 0 8px 25px rgba(0,0,0,0.3);
        text-align: center; z-index: 2000;
        border: 2px solid #e74c3c;
    `;
    reloginDiv.innerHTML = `
        <h3 style="color: #e74c3c; margin-top: 0;">🔒 세션 만료</h3>
        <p>안전을 위해 다시 로그인해 주세요</p>
        <button onclick="window.location.href='/login.html'" 
                style="background: #e74c3c; color: white; border: none; 
                       padding: 12px 24px; border-radius: 6px; 
                       cursor: pointer; font-size: 16px;">
            다시 로그인
        </button>
        <br><br>
        <button onclick="this.parentElement.remove()" 
                style="background: #95a5a6; color: white; border: none; 
                       padding: 8px 16px; border-radius: 4px; 
                       cursor: pointer; font-size: 14px;">
            나중에
        </button>
    `;
    
    // 배경 오버레이
    const overlay = document.createElement('div');
    overlay.style.cssText = `
        position: fixed; top: 0; left: 0; width: 100%; height: 100%;
        background: rgba(0,0,0,0.5); z-index: 1999;
    `;
    
    document.body.appendChild(overlay);
    document.body.appendChild(reloginDiv);
    
    overlay.onclick = () => {
        overlay.remove();
        reloginDiv.remove();
    };
}