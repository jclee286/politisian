// 지갑 관련 기능들

// PIN 모달 표시
function showPinModal() {
    const modal = document.getElementById('pin-modal');
    if (modal) {
        modal.style.display = 'flex';
        document.getElementById('pin-error').textContent = '';
        // PIN 입력 필드 초기화 및 첫 번째 필드에 포커스
        const pinInputs = document.querySelectorAll('.pin-digit-unlock');
        pinInputs.forEach(input => input.value = '');
        if (pinInputs.length > 0) {
            pinInputs[0].focus();
        }
    }
}

// PIN 모달 닫기
function closePinModal() {
    const modal = document.getElementById('pin-modal');
    if (modal) {
        modal.style.display = 'none';
    }
}

// 지갑 잠금 해제
function unlockWallet() {
    const pinInputs = document.querySelectorAll('.pin-digit-unlock');
    const pin = Array.from(pinInputs).map(input => input.value).join('');
    const errorElem = document.getElementById('pin-error');
    
    if (pin.length !== 6) {
        errorElem.textContent = '6자리 PIN을 모두 입력해주세요.';
        return;
    }
    
    console.log('PIN 검증 시도');
    
    fetch('/api/auth/verify-pin', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ pin: pin })
    })
    .then(response => {
        if (!response.ok) {
            return response.text().then(text => {
                throw new Error(text);
            });
        }
        return response.text();
    })
    .then(result => {
        console.log('✅ PIN 검증 성공:', result);
        closePinModal();
        
        // 지갑 잠금 해제 UI 업데이트
        const walletLocked = document.getElementById('wallet-locked');
        const walletUnlocked = document.getElementById('wallet-unlocked');
        if (walletLocked) walletLocked.style.display = 'none';
        if (walletUnlocked) walletUnlocked.style.display = 'block';
        
        showToast('지갑이 잠금 해제되었습니다! 🔓');
        
        // 지갑 데이터 로드
        loadWalletData();
    })
    .catch(error => {
        console.error('❌ PIN 검증 실패:', error);
        errorElem.textContent = 'PIN이 올바르지 않습니다.';
        
        // PIN 입력 필드 초기화
        pinInputs.forEach(input => input.value = '');
        if (pinInputs.length > 0) {
            pinInputs[0].focus();
        }
    });
}

// 지갑 잠금
function lockWallet() {
    const walletLocked = document.getElementById('wallet-locked');
    const walletUnlocked = document.getElementById('wallet-unlocked');
    if (walletLocked) walletLocked.style.display = 'block';
    if (walletUnlocked) walletUnlocked.style.display = 'none';
    
    showToast('지갑이 안전하게 잠겨졌습니다 🔒');
}

// 지갑 데이터 로드
function loadWalletData() {
    console.log('💰 지갑 데이터 로드 시작');
    // 현재는 프로필 로드와 동일한 기능
    loadUserProfile();
}

// 입출금 모달 관련
function showDepositModal() {
    const modal = document.getElementById('deposit-modal');
    if (modal) {
        modal.style.display = 'flex';
        loadDepositAddress();
    }
}

function closeDepositModal() {
    const modal = document.getElementById('deposit-modal');
    if (modal) modal.style.display = 'none';
}

function showWithdrawModal() {
    const modal = document.getElementById('withdraw-modal');
    if (modal) {
        modal.style.display = 'flex';
        updateAvailableBalance();
        
        // 출금 폼 초기화
        const form = document.getElementById('withdraw-form');
        if (form) form.reset();
        
        document.getElementById('withdraw-status').innerHTML = '';
    }
}

function updateAvailableBalance() {
    // 현재 잔액 표시 업데이트
    const tetherBalance = document.getElementById('available-tether')?.textContent || '0';
    const usdcBalance = document.getElementById('available-usdc')?.textContent || '0';
    
    const availableBalance = document.getElementById('available-balance-display');
    if (availableBalance) {
        availableBalance.innerHTML = `
            <strong>출금 가능 잔액:</strong><br>
            USDT: ${tetherBalance}<br>
            USDC: ${usdcBalance}
        `;
    }
}

function closeWithdrawModal() {
    const modal = document.getElementById('withdraw-modal');
    if (modal) modal.style.display = 'none';
}

// 입금 주소 로드
function loadDepositAddress() {
    console.log('📥 입금 주소 로드 시작');
    
    fetch('/api/wallet/address')
        .then(response => {
            if (!response.ok) {
                throw new Error('입금 주소 로드 실패');
            }
            return response.json();
        })
        .then(data => {
            console.log('✅ 입금 주소 로드 성공:', data);
            
            const addressElem = document.getElementById('deposit-address');
            if (addressElem && data.polygon_address) {
                addressElem.textContent = data.polygon_address;
                document.getElementById('copy-deposit-btn').style.display = 'inline-block';
            }
        })
        .catch(error => {
            console.error('❌ 입금 주소 로드 실패:', error);
            const addressElem = document.getElementById('deposit-address');
            if (addressElem) {
                addressElem.textContent = '입금 주소를 불러올 수 없습니다.';
            }
        });
}

// 입금 주소 복사
function copyDepositAddress() {
    const addressElem = document.getElementById('deposit-address');
    if (addressElem && addressElem.textContent) {
        copyToClipboard(addressElem.textContent, 'Polygon 입금 주소가 복사되었습니다! 📋');
    }
}

// 크레딧 사용 모달
function showCreditUsageModal() {
    const modal = document.getElementById('credit-usage-modal');
    if (modal) {
        modal.style.display = 'flex';
        loadAvailablePoliticiansForCredit();
        
        // 폼 초기화
        document.getElementById('credit-usage-form').reset();
        document.getElementById('credit-usage-status').innerHTML = '';
        
        // 현재 크레딧 수 표시
        const userProfile = getCurrentUserProfile();
        const availableCredits = userProfile ? userProfile.referral_credits : 0;
        document.getElementById('credit-amount').max = availableCredits;
        document.getElementById('credit-amount').value = Math.min(1, availableCredits);
    }
}

function closeCreditUsageModal() {
    const modal = document.getElementById('credit-usage-modal');
    if (modal) modal.style.display = 'none';
}

function loadAvailablePoliticiansForCredit() {
    const select = document.getElementById('credit-politician-select');
    if (!select) return;
    
    select.innerHTML = '<option value="">정치인을 선택하세요...</option>';
    
    fetch('/api/politisian/registered')
        .then(response => response.json())
        .then(politicians => {
            if (politicians && Array.isArray(politicians)) {
                const userProfile = getCurrentUserProfile();
                const alreadySelected = userProfile ? userProfile.politicians : [];
                
                politicians.forEach(politician => {
                    if (!alreadySelected.includes(politician.name)) {
                        const option = document.createElement('option');
                        option.value = politician.name;
                        option.textContent = `${politician.name} (${politician.party})`;
                        select.appendChild(option);
                    }
                });
            }
        })
        .catch(error => {
            console.error('정치인 목록 로드 실패:', error);
            select.innerHTML = '<option value="">목록 로드 실패</option>';
        });
}

function processCreditUsage() {
    const politician = document.getElementById('credit-politician-select').value;
    const creditAmount = parseInt(document.getElementById('credit-amount').value);
    const pin = document.getElementById('credit-pin').value;
    const statusElem = document.getElementById('credit-usage-status');

    // 입력값 검증
    if (!politician) {
        statusElem.innerHTML = '<span style="color: #e74c3c;">정치인을 선택하세요.</span>';
        return;
    }

    if (!pin || pin.length !== 6) {
        statusElem.innerHTML = '<span style="color: #e74c3c;">6자리 PIN을 입력하세요.</span>';
        return;
    }

    statusElem.innerHTML = '<span style="color: #f39c12;">크레딧 사용 처리 중...</span>';

    const usageData = {
        politician: politician,
        credit_amount: creditAmount,
        pin: pin
    };

    fetch('/api/user/use-credit', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(usageData)
    })
    .then(response => {
        if (!response.ok) {
            return response.text().then(text => {
                throw new Error(text);
            });
        }
        return response.json();
    })
    .then(data => {
        statusElem.innerHTML = '<span style="color: #28a745;">✅ 크레딧이 성공적으로 사용되었습니다!</span>';
        
        // 3초 후 모달 닫기 및 데이터 새로고침
        setTimeout(() => {
            closeCreditUsageModal();
            loadUserProfile();
            showToast(`🎉 축하합니다! ${politician} 코인 100개를 받았어요!`);
        }, 3000);
    })
    .catch(error => {
        console.error('크레딧 사용 실패:', error);
        statusElem.innerHTML = `<span style="color: #e74c3c;">사용 실패: ${error.message}</span>`;
    });
}