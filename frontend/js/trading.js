// 거래 관련 기능들

// 거래 데이터 로드
function loadTradingData() {
    console.log('📊 거래 데이터 로드 시작');
    loadPoliticianPrices();
    loadMyOrders();
    loadPoliticianSelectOptions();
}

// 정치인 가격 정보 로드
function loadPoliticianPrices() {
    console.log('💰 정치인 가격 정보 로드 시작');
    
    fetch('/api/trading/prices')
        .then(response => {
            if (!response.ok) {
                // 가격 정보는 선택적 기능이므로 에러를 조용히 처리
                console.warn('가격 정보 로드 실패 (선택적 기능):', response.status);
                return null;
            }
            return response.json();
        })
        .then(prices => {
            if (prices) {
                console.log('✅ 정치인 가격 정보 로드 성공:', prices);
                displayPriceRankings(prices);
            } else {
                console.log('가격 정보를 표시하지 않음 (API 미지원)');
                // 가격 랭킹 섹션 숨기기
                const priceSection = document.getElementById('price-rankings');
                if (priceSection) {
                    priceSection.style.display = 'none';
                }
            }
        })
        .catch(error => {
            console.warn('가격 정보 로드 실패 (선택적 기능):', error);
            // 에러를 조용히 처리하고 UI에서 해당 섹션을 숨김
            const priceSection = document.getElementById('price-rankings');
            if (priceSection) {
                priceSection.style.display = 'none';
            }
        });
}

// 가격 랭킹 표시
function displayPriceRankings(prices) {
    const priceList = document.getElementById('price-rankings-list');
    if (!priceList || !prices || prices.length === 0) {
        return;
    }
    
    priceList.innerHTML = '';
    
    // 가격 순으로 정렬
    const sortedPrices = [...prices].sort((a, b) => (b.current_price || 0) - (a.current_price || 0));
    
    sortedPrices.slice(0, 10).forEach((item, index) => {
        const li = document.createElement('li');
        const changeIcon = (item.price_change || 0) >= 0 ? '📈' : '📉';
        const changeColor = (item.price_change || 0) >= 0 ? '#28a745' : '#dc3545';
        
        li.innerHTML = `
            <div style="display: flex; justify-content: space-between; align-items: center;">
                <div>
                    <strong>${index + 1}. ${item.politician_name}</strong>
                    <br><small style="color: ${changeColor};">${changeIcon} ${item.price_change || 0}%</small>
                </div>
                <div style="text-align: right;">
                    <div style="font-weight: bold;">${(item.current_price || 0).toFixed(2)} USDT</div>
                    <small>거래량: ${item.volume || 0}</small>
                </div>
            </div>
        `;
        
        // 클릭 시 거래 선택
        li.style.cursor = 'pointer';
        li.onclick = () => selectPoliticianForTrade(item.politician_id, item.politician_name);
        
        priceList.appendChild(li);
    });
}

// 거래용 정치인 선택
function selectPoliticianForTrade(politicianId, name) {
    const select = document.getElementById('trade-politician');
    if (select) {
        select.value = politicianId;
        updateTradeSummary();
    }
    
    console.log(`거래 대상 선택: ${name} (${politicianId})`);
}

// 거래용 정치인 선택 옵션 로드
function loadPoliticianSelectOptions() {
    const select = document.getElementById('trade-politician');
    if (!select) return;
    
    select.innerHTML = '<option value="">정치인을 선택하세요...</option>';
    
    if (typeof window.allPoliticiansData === 'object') {
        Object.entries(window.allPoliticiansData).forEach(([id, politician]) => {
            const option = document.createElement('option');
            option.value = id;
            option.textContent = `${politician.name} (${politician.party || '무소속'})`;
            select.appendChild(option);
        });
    }
}

// 내 주문 목록 로드
function loadMyOrders() {
    console.log('📋 내 주문 목록 로드 시작');
    
    fetch('/api/trading/my-orders')
        .then(response => {
            if (!response.ok) {
                throw new Error('주문 목록 로드 실패');
            }
            return response.json();
        })
        .then(orders => {
            console.log('✅ 내 주문 목록 로드 성공:', orders);
            displayMyOrders(orders);
        })
        .catch(error => {
            console.error('❌ 내 주문 목록 로드 실패:', error);
            const ordersElem = document.getElementById('my-orders');
            if (ordersElem) {
                ordersElem.innerHTML = '<div style="color: #e74c3c;">주문 목록을 불러올 수 없습니다.</div>';
            }
        });
}

// 내 주문 목록 표시
function displayMyOrders(orders) {
    const ordersElem = document.getElementById('my-orders');
    if (!ordersElem) return;
    
    if (!orders || orders.length === 0) {
        ordersElem.innerHTML = '<div style="color: #666;">활성 주문이 없습니다.</div>';
        return;
    }
    
    ordersElem.innerHTML = '';
    
    orders.forEach(order => {
        const orderDiv = document.createElement('div');
        orderDiv.style.cssText = `
            background: #f8f9fa; padding: 10px; margin-bottom: 10px; 
            border-radius: 4px; border-left: 4px solid ${order.order_type === 'buy' ? '#28a745' : '#dc3545'};
        `;
        
        const typeText = order.order_type === 'buy' ? '구매' : '판매';
        const typeIcon = order.order_type === 'buy' ? '📈' : '📉';
        
        orderDiv.innerHTML = `
            <div style="display: flex; justify-content: space-between; align-items: center;">
                <div>
                    <strong>${typeIcon} ${typeText}</strong> - ${order.politician_name || order.politician}
                    <br><small>수량: ${order.quantity}개 | 가격: ${order.price} ${order.currency || 'USDT'}</small>
                    <br><small>상태: ${order.status || '대기중'}</small>
                </div>
                <button onclick="cancelOrder('${order.id}')" 
                        style="background: #dc3545; color: white; border: none; padding: 5px 10px; border-radius: 4px; cursor: pointer;">
                    취소
                </button>
            </div>
        `;
        
        ordersElem.appendChild(orderDiv);
    });
}

// 거래 요약 업데이트
function updateTradeSummary() {
    const politician = document.getElementById('trade-politician').value;
    const quantity = parseInt(document.getElementById('trade-quantity').value) || 0;
    const price = parseFloat(document.getElementById('trade-price').value) || 0;
    const orderType = document.querySelector('input[name="order-type"]:checked')?.value || 'buy';
    const currency = document.querySelector('input[name="currency"]:checked')?.value || 'USDT';
    
    const summaryElem = document.getElementById('trade-summary');
    if (!summaryElem) return;
    
    if (!politician || quantity <= 0 || price <= 0) {
        summaryElem.textContent = '정치인과 수량, 가격을 선택하세요';
        return;
    }
    
    const total = quantity * price;
    const politicianName = getPoliticianNameById(politician);
    const typeText = orderType === 'buy' ? '구매' : '판매';
    
    summaryElem.innerHTML = `
        <strong>${typeText}:</strong> ${politicianName} ${quantity}개<br>
        <strong>단가:</strong> ${price} ${currency}<br>
        <strong>총액:</strong> ${total.toFixed(2)} ${currency}
    `;
}

// 거래 주문 등록
function placeTradeOrder() {
    const politician = document.getElementById('trade-politician').value;
    const quantity = parseInt(document.getElementById('trade-quantity').value);
    const price = parseFloat(document.getElementById('trade-price').value);
    const orderType = document.querySelector('input[name="order-type"]:checked')?.value;
    const currency = document.querySelector('input[name="currency"]:checked')?.value || 'USDT';
    
    // 입력값 검증
    if (!politician) {
        showToast('정치인을 선택해주세요.', 'error');
        return;
    }
    
    if (!quantity || quantity <= 0) {
        showToast('올바른 수량을 입력해주세요.', 'error');
        return;
    }
    
    if (!price || price <= 0) {
        showToast('올바른 가격을 입력해주세요.', 'error');
        return;
    }
    
    if (!orderType) {
        showToast('주문 유형을 선택해주세요.', 'error');
        return;
    }
    
    const orderData = {
        politician: politician,
        quantity: quantity,
        price: price,
        order_type: orderType,
        currency: currency
    };
    
    console.log('거래 주문 데이터:', orderData);
    
    const statusElem = document.getElementById('trade-status');
    if (statusElem) {
        statusElem.innerHTML = '<span style="color: #f39c12;">주문 등록 중...</span>';
    }
    
    fetch('/api/trading/place-order', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(orderData)
    })
    .then(response => {
        if (!response.ok) {
            return response.text().then(text => {
                throw new Error(text);
            });
        }
        return response.json();
    })
    .then(result => {
        console.log('✅ 거래 주문 성공:', result);
        
        if (statusElem) {
            statusElem.innerHTML = '<span style="color: #28a745;">✅ 주문이 등록되었습니다!</span>';
        }
        
        showToast('거래 주문이 성공적으로 등록되었습니다! 📊');
        
        // 폼 초기화
        document.getElementById('trade-form').reset();
        document.getElementById('trade-summary').textContent = '정치인과 수량을 선택하세요';
        
        // 주문 목록 새로고침
        setTimeout(() => {
            loadMyOrders();
            if (statusElem) {
                statusElem.innerHTML = '';
            }
        }, 2000);
        
        // 프로필 새로고침 (잔액 업데이트)
        loadUserProfile();
    })
    .catch(error => {
        console.error('❌ 거래 주문 실패:', error);
        
        if (statusElem) {
            statusElem.innerHTML = `<span style="color: #e74c3c;">주문 실패: ${error.message}</span>`;
        }
        
        showToast(`주문 실패: ${error.message}`, 'error');
        
        setTimeout(() => {
            if (statusElem) {
                statusElem.innerHTML = '';
            }
        }, 5000);
    });
}

// 주문 취소
function cancelOrder(orderId) {
    if (!confirm('정말로 이 주문을 취소하시겠습니까?')) {
        return;
    }
    
    console.log('주문 취소 시도:', orderId);
    
    fetch(`/api/trading/cancel-order/${orderId}`, {
        method: 'DELETE'
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
        console.log('✅ 주문 취소 성공:', result);
        showToast('주문이 취소되었습니다.');
        
        // 주문 목록 새로고침
        loadMyOrders();
        
        // 프로필 새로고침 (잔액 업데이트)
        loadUserProfile();
    })
    .catch(error => {
        console.error('❌ 주문 취소 실패:', error);
        showToast(`주문 취소 실패: ${error.message}`, 'error');
    });
}

// 거래 폼 이벤트 설정
function setupTradingEvents() {
    const tradeForm = document.getElementById('trade-form');
    if (tradeForm) {
        tradeForm.addEventListener('submit', function(event) {
            event.preventDefault();
            placeTradeOrder();
        });
    }
    
    // 거래 요약 자동 업데이트
    const tradeInputs = ['trade-politician', 'trade-quantity', 'trade-price'];
    tradeInputs.forEach(id => {
        const elem = document.getElementById(id);
        if (elem) {
            elem.addEventListener('input', updateTradeSummary);
            elem.addEventListener('change', updateTradeSummary);
        }
    });
    
    // 라디오 버튼 변경 시 요약 업데이트
    const radioButtons = document.querySelectorAll('input[name="order-type"], input[name="currency"]');
    radioButtons.forEach(radio => {
        radio.addEventListener('change', updateTradeSummary);
    });
}