// 대시보드 UI 업데이트 및 관리

// UI 업데이트 함수
function updateDashboardUI(data) {
    try {
        console.log('🎨 UI 업데이트 시작');
        console.log('프로필 데이터 전체:', data);
        
        // 안전한 요소 접근
        if (walletAddressElem) {
            setWalletAddress(data.wallet || '주소 없음');
        }
        
        // 총 코인 표시 업데이트
        if (totalCoinsElem) {
            const totalCoins = data.total_coins || data.balance || 0;
            totalCoinsElem.textContent = `${totalCoins}개`;
        }
        
        // USDT 잔액 업데이트
        const tetherBalanceElem = document.getElementById('tether-balance');
        const availableTetherElem = document.getElementById('available-tether');
        if (tetherBalanceElem && availableTetherElem) {
            const tetherBalance = data.usdt_balance || 0;
            const frozenTether = (data.escrow_account && data.escrow_account.frozen_usdt_balance) || 0;
            const availableTether = tetherBalance - frozenTether;
            
            tetherBalanceElem.textContent = `${tetherBalance}`;
            availableTetherElem.textContent = `${availableTether}`;
            
            if (frozenTether > 0) {
                availableTetherElem.style.color = '#ffc107';
                availableTetherElem.title = `총 ${tetherBalance} USDT 중 ${frozenTether} USDT 동결됨`;
            } else {
                availableTetherElem.style.color = '#28a745';
                availableTetherElem.title = '';
            }
        }
        
        // USDC 잔액 업데이트
        const usdcBalanceElem = document.getElementById('usdc-balance');
        const availableUsdcElem = document.getElementById('available-usdc');
        if (usdcBalanceElem && availableUsdcElem) {
            const usdcBalance = data.usdc_balance || 0;
            const frozenUsdc = (data.escrow_account && data.escrow_account.frozen_usdc_balance) || 0;
            const availableUsdc = usdcBalance - frozenUsdc;
            
            usdcBalanceElem.textContent = `${usdcBalance}`;
            availableUsdcElem.textContent = `${availableUsdc}`;
            
            if (frozenUsdc > 0) {
                availableUsdcElem.style.color = '#ffc107';
                availableUsdcElem.title = `총 ${usdcBalance} USDC 중 ${frozenUsdc} USDC 동결됨`;
            } else {
                availableUsdcElem.style.color = '#28a745';
                availableUsdcElem.title = '';
            }
        }

        // 추천 크레딧 표시
        const referralCreditsElem = document.getElementById('referral-credits');
        if (referralCreditsElem) {
            const credits = data.referral_credits || 0;
            referralCreditsElem.textContent = credits;
            console.log('추천 크레딧 업데이트:', credits);
            
            if (credits > 0) {
                referralCreditsElem.parentElement.style.backgroundColor = '#d4edda';
                referralCreditsElem.parentElement.style.borderColor = '#c3e6cb';
                
                document.getElementById('credit-usage-section').style.display = 'block';
            }
        }
        
        // 추천 링크 생성
        generateReferralLink(data.wallet);

        // 정치인별 코인 보유량 표시
        if (politicianCoinsListElem && data.politician_coins) {
            console.log('정치인별 코인 데이터:', data.politician_coins);
            
            const coinEntries = Object.entries(data.politician_coins);
            
            if (coinEntries.length > 0) {
                politicianCoinsListElem.innerHTML = '';
                coinEntries.forEach(([politicianId, coins]) => {
                    const li = document.createElement('li');
                    const politicianName = getPoliticianNameById(politicianId);
                    li.innerHTML = `${politicianName} <span style="float: right; color: #28a745; font-weight: bold;">${coins} 코인</span>`;
                    politicianCoinsListElem.appendChild(li);
                });
            } else {
                // 코인이 없는 경우 안내 메시지만 표시
                politicianCoinsListElem.innerHTML = `
                    <li class="text-center py-8 text-gray-500">
                        <div class="text-6xl mb-4">💰</div>
                        <div class="font-medium mb-2">보유 중인 정치인 코인이 없습니다</div>
                        <div class="text-sm text-gray-400">거래를 통해 정치인 코인을 구매해보세요</div>
                    </li>
                `;
            }
        } else if (politicianCoinsListElem) {
            politicianCoinsListElem.innerHTML = '<li>코인 정보를 불러올 수 없습니다.</li>';
        }
        
        // 로그인 후 자동으로 지갑 해제
        console.log('🔓 로그인 후 자동 지갑 해제');
        const walletLocked = document.getElementById('wallet-locked');
        const walletUnlocked = document.getElementById('wallet-unlocked');
        if (walletLocked) walletLocked.style.display = 'none';
        if (walletUnlocked) walletUnlocked.style.display = 'block';
        
        console.log('✅ UI 업데이트 완료');
        
    } catch (error) {
        console.error('❌ updateDashboardUI 오류:', error);
    }
}

// 지갑 주소 설정
function setWalletAddress(address) {
    if (walletAddressElem) {
        walletAddressElem.textContent = address;
        
        const copyButton = document.getElementById('copy-wallet-btn');
        if (copyButton && address !== '불러오는 중...' && address !== '지갑 주소 없음') {
            copyButton.style.display = 'inline-block';
        }
    }
}

// 지갑 주소 복사
function copyWalletAddress() {
    const address = walletAddressElem?.textContent;
    if (address && address !== '불러오는 중...' && address !== '지갑 주소 없음') {
        copyToClipboard(address, '지갑 주소가 복사되었습니다! 📋');
    } else {
        showToast('복사할 수 있는 지갑 주소가 없습니다.', 'error');
    }
}

// 추천 링크 생성
function generateReferralLink(walletAddress) {
    if (!walletAddress || walletAddress === '지갑 주소 없음') {
        console.log('지갑 주소가 없어 추천 링크를 생성할 수 없음');
        return;
    }
    
    // 내부 ID가 아닌 실제 Polygon 지갑 주소 사용
    const referralCode = walletAddress.length > 10 ? walletAddress.substring(0, 10) : walletAddress;
    const baseUrl = window.location.origin;
    const referralUrl = `${baseUrl}/signup.html?ref=${encodeURIComponent(referralCode)}`;
    
    const referralLinkElem = document.getElementById('referral-link');
    const copyReferralBtn = document.getElementById('copy-referral-btn');
    const shareReferralBtn = document.getElementById('share-referral-btn');
    
    if (referralLinkElem) {
        referralLinkElem.textContent = referralUrl;
    }
    
    if (copyReferralBtn) {
        copyReferralBtn.style.display = 'inline-block';
    }
    
    if (shareReferralBtn) {
        shareReferralBtn.style.display = 'inline-block';
    }
    
    console.log('추천 링크 생성됨:', referralUrl);
}

// 추천 링크 복사
function copyReferralLink() {
    const referralLinkElem = document.getElementById('referral-link');
    if (referralLinkElem && referralLinkElem.textContent) {
        copyToClipboard(referralLinkElem.textContent, '추천 링크가 복사되었습니다! 친구들에게 공유해보세요 🎉');
    }
}

// 추천 링크 공유
function shareReferralLink() {
    const referralLinkElem = document.getElementById('referral-link');
    const referralUrl = referralLinkElem?.textContent;
    
    if (!referralUrl) {
        showToast('공유할 수 있는 추천 링크가 없습니다.', 'error');
        return;
    }
    
    const shareText = '🪙 나의 공화국에서 정치인 코인을 받아보세요!\n\n이 링크로 가입하면 특별 혜택이 있어요 ✨';
    
    if (navigator.share) {
        navigator.share({
            title: '나의 공화국 추천',
            text: shareText,
            url: referralUrl
        }).then(() => {
            console.log('추천 링크 공유 성공');
            showToast('추천 링크가 공유되었습니다! 🎉');
        }).catch((error) => {
            console.log('네이티브 공유 실패, 복사로 대체:', error);
            copyToClipboard(`${shareText}\n\n${referralUrl}`, '추천 링크와 메시지가 복사되었습니다! 📋');
        });
    } else {
        copyToClipboard(`${shareText}\n\n${referralUrl}`, '추천 링크와 메시지가 복사되었습니다! 📋');
    }
}