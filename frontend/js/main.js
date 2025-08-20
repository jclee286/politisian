// 전역 변수
let allPoliticiansData = {};
let currentUserProfileData = null;

// DOM 요소 참조
let walletAddressElem, politicianCoinsListElem, totalCoinsElem, loginButton;
let copyStatus, proposalsListElem, registeredPoliticiansListElem;
let searchPoliticiansInput, proposeForm, proposeStatus;

// DOM 초기화
document.addEventListener('DOMContentLoaded', function() {
    console.log('🏠 완전한 대시보드 페이지 로드됨 (v2.0)');
    
    // 완전히 안전한 요소 참조
    walletAddressElem = document.getElementById('wallet-address');
    politicianCoinsListElem = document.getElementById('politician-coins-list');
    totalCoinsElem = document.getElementById('total-coins');
    loginButton = document.getElementById('login-button');
    copyStatus = document.getElementById('copy-status');
    proposalsListElem = document.getElementById('proposals-list');
    registeredPoliticiansListElem = document.getElementById('registered-politicians-list');
    searchPoliticiansInput = document.getElementById('search-politicians');
    proposeForm = document.getElementById('propose-form');
    proposeStatus = document.getElementById('propose-status');

    console.log('🍪 현재 쿠키:', document.cookie);

    // 초기 데이터 로드
    loadUserProfile();
    
    // 이벤트 리스너 등록
    setupEventListeners();
});

// 이벤트 리스너 설정
function setupEventListeners() {
    // 검색 기능
    if (searchPoliticiansInput) {
        searchPoliticiansInput.addEventListener('input', function(e) {
            searchPoliticians(e.target.value);
        });
    }
    
    // 제안 폼
    if (proposeForm) {
        proposeForm.addEventListener('submit', handleProposeSubmit);
    }
    
    // 제안 목록 투표
    if (proposalsListElem) {
        proposalsListElem.addEventListener('click', handleProposalVote);
    }
    
    // PIN 입력 이벤트
    setupPinInputEvents();
    
    // 거래 폼 이벤트
    setupTradingEvents();
    
    // 모달 폼 이벤트
    setupModalEvents();
}

// 모달 이벤트 설정
function setupModalEvents() {
    // 초기 코인 받기 폼
    const claimCoinsForm = document.getElementById('claim-coins-form');
    if (claimCoinsForm) {
        claimCoinsForm.addEventListener('submit', function(event) {
            event.preventDefault();
            processClaimCoins();
        });
    }
    
    // 크레딧 사용 폼
    const creditUsageForm = document.getElementById('credit-usage-form');
    if (creditUsageForm) {
        creditUsageForm.addEventListener('submit', function(event) {
            event.preventDefault();
            processCreditUsage();
        });
    }
}

// PIN 입력 이벤트 설정
function setupPinInputEvents() {
    const pinInputs = document.querySelectorAll('.pin-digit-unlock');
    pinInputs.forEach((input, index) => {
        input.addEventListener('input', function(e) {
            if (e.target.value.length === 1 && index < pinInputs.length - 1) {
                pinInputs[index + 1].focus();
            }
        });
        
        input.addEventListener('keydown', function(e) {
            if (e.key === 'Backspace' && e.target.value === '' && index > 0) {
                pinInputs[index - 1].focus();
            }
        });
    });
}

// 유틸리티 함수들
function showError(message) {
    const errorDiv = document.createElement('div');
    errorDiv.style.cssText = `
        position: fixed; top: 20px; right: 20px; 
        background: #e74c3c; color: white; 
        padding: 15px 20px; border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.3);
        z-index: 2000; font-size: 14px;
        max-width: 300px;
    `;
    errorDiv.textContent = message;
    document.body.appendChild(errorDiv);
    
    setTimeout(() => errorDiv.remove(), 5000);
}

function showToast(message, type = 'success') {
    const toast = document.createElement('div');
    const bgColor = type === 'error' ? '#e74c3c' : '#28a745';
    
    toast.style.cssText = `
        position: fixed; top: 20px; right: 20px; 
        background: ${bgColor}; color: white; 
        padding: 15px 20px; border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.3);
        z-index: 2000; font-size: 14px;
        max-width: 300px;
    `;
    toast.textContent = message;
    document.body.appendChild(toast);
    
    setTimeout(() => toast.remove(), 4000);
}

function saveBackupData(walletAddress, userInfo) {
    localStorage.setItem('backup_wallet_address', walletAddress);
    localStorage.setItem('backup_user_info', JSON.stringify(userInfo));
}

function getCurrentUserProfile() {
    return currentUserProfileData;
}

function getPoliticianNameById(politicianId) {
    if (typeof allPoliticiansData === 'object' && allPoliticiansData[politicianId]) {
        return allPoliticiansData[politicianId].name || politicianId;
    }
    return politicianId;
}

// 복사 기능
function copyToClipboard(text, successMessage = '복사되었습니다') {
    if (navigator.clipboard && window.isSecureContext) {
        navigator.clipboard.writeText(text).then(() => {
            showToast(successMessage);
        }).catch(() => {
            fallbackCopy(text, successMessage);
        });
    } else {
        fallbackCopy(text, successMessage);
    }
}

function fallbackCopy(text, successMessage) {
    const textArea = document.createElement('textarea');
    textArea.value = text;
    textArea.style.position = 'fixed';
    textArea.style.left = '-999999px';
    textArea.style.top = '-999999px';
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    
    try {
        document.execCommand('copy');
        showToast(successMessage);
    } catch (err) {
        console.error('복사 실패:', err);
        showToast('복사에 실패했습니다. 수동으로 복사해주세요.', 'error');
    }
    
    document.body.removeChild(textArea);
}