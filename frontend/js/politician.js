// 정치인 관련 기능들

// 제안 목록 로드
function loadProposals() {
    console.log('📋 제안 목록 로드 시작');
    
    fetch('/api/politisian/list')
        .then(response => {
            if (!response.ok) {
                throw new Error(`API 오류: ${response.status}`);
            }
            return response.json();
        })
        .then(proposals => {
            console.log('✅ 제안 목록 로드 성공:', proposals);
            
            if (!proposalsListElem) {
                console.warn('제안 목록 요소를 찾을 수 없음');
                return;
            }
            
            if (!proposals || proposals.length === 0) {
                proposalsListElem.innerHTML = '<li>진행 중인 제안이 없습니다.</li>';
                return;
            }
            
            proposalsListElem.innerHTML = '';
            proposals.forEach(proposal => {
                const li = document.createElement('li');
                li.innerHTML = `
                    <div style="display: flex; justify-content: space-between; align-items: center;">
                        <div>
                            <strong>${proposal.politician.name}</strong> (${proposal.politician.party})
                            <br><small>지역: ${proposal.politician.region || '미정'}</small>
                            <br><small>찬성: ${proposal.yes_votes || 0}표, 반대: ${proposal.no_votes || 0}표</small>
                        </div>
                        <div>
                            <button class="button vote-button approve" onclick="voteOnProposal('${proposal.id}', true)">
                                👍 찬성
                            </button>
                            <button class="button vote-button reject" onclick="voteOnProposal('${proposal.id}', false)">
                                👎 반대
                            </button>
                        </div>
                    </div>
                `;
                proposalsListElem.appendChild(li);
            });
        })
        .catch(error => {
            console.error('❌ 제안 목록 로드 실패:', error);
            if (proposalsListElem) {
                proposalsListElem.innerHTML = '<li class="error-message">제안 목록을 불러올 수 없습니다.</li>';
            }
        });
}

// 등록된 정치인 목록 로드
function loadRegisteredPoliticians() {
    console.log('🏛️ 등록된 정치인 목록 로드 시작');
    
    fetch('/api/politisian/registered')
        .then(response => {
            if (!response.ok) {
                throw new Error(`API 오류: ${response.status}`);
            }
            return response.json();
        })
        .then(politicians => {
            console.log('✅ 등록된 정치인 목록 로드 성공:', politicians);
            
            if (!registeredPoliticiansListElem) {
                console.warn('등록된 정치인 목록 요소를 찾을 수 없음');
                return;
            }
            
            // 전역 데이터에 저장 (검색용)
            window.allPoliticiansData = {};
            if (politicians && Array.isArray(politicians)) {
                politicians.forEach(politician => {
                    window.allPoliticiansData[politician.name] = politician;
                });
            }
            
            displayPoliticians(politicians);
        })
        .catch(error => {
            console.error('❌ 등록된 정치인 목록 로드 실패:', error);
            if (registeredPoliticiansListElem) {
                registeredPoliticiansListElem.innerHTML = '<li class="error-message">등록된 정치인 목록을 불러올 수 없습니다.</li>';
            }
        });
}

// 정치인 목록 표시
function displayPoliticians(politicians) {
    if (!registeredPoliticiansListElem) return;
    
    if (!politicians || politicians.length === 0) {
        registeredPoliticiansListElem.innerHTML = '<li>등록된 정치인이 없습니다.</li>';
        return;
    }
    
    registeredPoliticiansListElem.innerHTML = '';
    politicians.forEach(politician => {
        const li = document.createElement('li');
        li.innerHTML = `
            <div style="display: flex; justify-content: space-between; align-items: center;">
                <div>
                    <strong>${politician.name}</strong> (${politician.party || '무소속'})
                    <br><small>지역: ${politician.region || '전국구'}</small>
                    <br><small>총 발행: ${politician.total_coin_supply?.toLocaleString() || '0'}개</small>
                    <br><small>남은 코인: ${politician.remaining_coins?.toLocaleString() || '0'}개</small>
                </div>
                <div>
                    <button onclick="selectPoliticianForTrade('${politician.name}', '${politician.name}')" 
                            style="background: #007bff; color: white; border: none; padding: 5px 10px; border-radius: 4px; cursor: pointer; font-size: 12px;">
                        📊 거래
                    </button>
                </div>
            </div>
        `;
        registeredPoliticiansListElem.appendChild(li);
    });
}

// 정치인 검색
function searchPoliticians(searchTerm) {
    if (!searchTerm || !window.allPoliticiansData) {
        // 검색어가 없으면 전체 목록 표시
        const allPoliticians = Object.values(window.allPoliticiansData);
        displayPoliticians(allPoliticians);
        return;
    }
    
    const filteredPoliticians = Object.values(window.allPoliticiansData).filter(politician => {
        const nameMatch = politician.name.toLowerCase().includes(searchTerm.toLowerCase());
        const regionMatch = politician.region && politician.region.toLowerCase().includes(searchTerm.toLowerCase());
        const partyMatch = politician.party && politician.party.toLowerCase().includes(searchTerm.toLowerCase());
        
        return nameMatch || regionMatch || partyMatch;
    });
    
    displayPoliticians(filteredPoliticians);
}

// 제안 투표
function voteOnProposal(proposalId, vote) {
    console.log(`투표 시도: 제안 ID ${proposalId}, 투표 ${vote ? '찬성' : '반대'}`);
    
    fetch(`/api/proposals/${proposalId}/vote`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ vote: vote })
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
        console.log('✅ 투표 성공:', result);
        showToast(`투표가 완료되었습니다! (${vote ? '찬성' : '반대'})`);
        
        // 제안 목록 새로고침
        setTimeout(() => {
            loadProposals();
        }, 1000);
    })
    .catch(error => {
        console.error('❌ 투표 실패:', error);
        showToast(`투표 실패: ${error.message}`, 'error');
    });
}

// 새 정치인 제안
function handleProposeSubmit(event) {
    event.preventDefault();
    
    const name = document.getElementById('name').value.trim();
    const region = document.getElementById('region').value.trim();
    const party = document.getElementById('party').value.trim();
    const introUrl = document.getElementById('intro-url').value.trim();
    
    if (!name || !party) {
        showToast('이름과 정당은 필수 입력사항입니다.', 'error');
        return;
    }
    
    const proposalData = {
        name: name,
        region: region || '',
        party: party,
        introUrl: introUrl || ''
    };
    
    console.log('새 정치인 제안 데이터:', proposalData);
    
    proposeStatus.textContent = '제안 중...';
    proposeStatus.style.color = '#f39c12';
    
    fetch('/api/politisian/propose', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(proposalData)
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
        console.log('✅ 정치인 제안 성공:', result);
        proposeStatus.textContent = '제안이 성공적으로 등록되었습니다!';
        proposeStatus.style.color = '#28a745';
        
        // 폼 초기화
        proposeForm.reset();
        
        // 제안 목록 새로고침
        setTimeout(() => {
            loadProposals();
            proposeStatus.textContent = '';
        }, 3000);
    })
    .catch(error => {
        console.error('❌ 정치인 제안 실패:', error);
        proposeStatus.textContent = `제안 실패: ${error.message}`;
        proposeStatus.style.color = '#e74c3c';
        
        setTimeout(() => {
            proposeStatus.textContent = '';
        }, 5000);
    });
}

// 제안 목록 이벤트 핸들러
function handleProposalVote(event) {
    if (event.target.classList.contains('vote-button')) {
        const proposalId = event.target.getAttribute('onclick').match(/'([^']+)'/)[1];
        const isApprove = event.target.classList.contains('approve');
        voteOnProposal(proposalId, isApprove);
    }
}