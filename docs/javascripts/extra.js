// AICode Manager 문서 사이트 추가 JavaScript

// 문서 로드 완료 후 실행
document.addEventListener('DOMContentLoaded', function() {
    // 코드 블록에 복사 기능 개선
    enhanceCodeBlocks();
    
    // 외부 링크 처리
    handleExternalLinks();
    
    // 테이블 반응형 처리
    makeTablesResponsive();
    
    // 검색 결과 개선
    enhanceSearch();
    
    // 다크모드 토글 이벤트
    handleThemeToggle();
    
    // 네비게이션 개선
    enhanceNavigation();
    
    // 피드백 시스템
    initializeFeedback();
});

// 코드 블록 개선
function enhanceCodeBlocks() {
    const codeBlocks = document.querySelectorAll('pre code');
    
    codeBlocks.forEach(function(block) {
        // 언어 표시 추가
        const language = block.className.match(/language-(\w+)/);
        if (language) {
            const languageLabel = document.createElement('span');
            languageLabel.className = 'code-language';
            languageLabel.textContent = language[1].toUpperCase();
            languageLabel.style.cssText = `
                position: absolute;
                top: 0.5rem;
                right: 3rem;
                font-size: 0.7rem;
                background: rgba(0,0,0,0.3);
                color: white;
                padding: 0.2rem 0.5rem;
                border-radius: 0.2rem;
                z-index: 1;
            `;
            
            const pre = block.parentElement;
            if (pre.tagName === 'PRE') {
                pre.style.position = 'relative';
                pre.appendChild(languageLabel);
            }
        }
        
        // 행 번호 추가 (선택사항)
        if (block.textContent.split('\n').length > 5) {
            addLineNumbers(block);
        }
    });
}

// 행 번호 추가 함수
function addLineNumbers(codeBlock) {
    const lines = codeBlock.textContent.split('\n');
    const numberedLines = lines.map((line, index) => {
        const lineNumber = String(index + 1).padStart(2, ' ');
        return `<span class="line-number">${lineNumber}</span>${line}`;
    }).join('\n');
    
    codeBlock.innerHTML = numberedLines;
    
    // 행 번호 스타일
    const style = document.createElement('style');
    style.textContent = `
        .line-number {
            display: inline-block;
            width: 2rem;
            margin-right: 1rem;
            color: #666;
            text-align: right;
            user-select: none;
            border-right: 1px solid #ddd;
            padding-right: 0.5rem;
        }
        [data-md-color-scheme="slate"] .line-number {
            color: #888;
            border-color: #444;
        }
    `;
    document.head.appendChild(style);
}

// 외부 링크 처리
function handleExternalLinks() {
    const links = document.querySelectorAll('a[href^="http"]');
    
    links.forEach(function(link) {
        // 현재 도메인이 아닌 경우 외부 링크로 처리
        if (!link.href.includes(window.location.hostname)) {
            link.setAttribute('target', '_blank');
            link.setAttribute('rel', 'noopener noreferrer');
            
            // 외부 링크 아이콘 추가
            const icon = document.createElement('span');
            icon.innerHTML = ' ↗';
            icon.style.fontSize = '0.8em';
            icon.style.opacity = '0.7';
            link.appendChild(icon);
        }
    });
}

// 테이블 반응형 처리
function makeTablesResponsive() {
    const tables = document.querySelectorAll('table');
    
    tables.forEach(function(table) {
        // 테이블을 스크롤 가능한 컨테이너로 감싸기
        const wrapper = document.createElement('div');
        wrapper.className = 'table-wrapper';
        wrapper.style.cssText = `
            overflow-x: auto;
            margin: 1rem 0;
            border: 1px solid var(--md-default-fg-color--lighter);
            border-radius: 0.2rem;
        `;
        
        table.parentNode.insertBefore(wrapper, table);
        wrapper.appendChild(table);
        
        // 모바일에서 테이블 힌트 추가
        if (window.innerWidth < 768) {
            const hint = document.createElement('div');
            hint.textContent = '← 좌우로 스크롤하여 더 보기';
            hint.style.cssText = `
                font-size: 0.8rem;
                color: #666;
                text-align: center;
                padding: 0.5rem;
                background: #f5f5f5;
                border-bottom: 1px solid var(--md-default-fg-color--lighter);
            `;
            wrapper.insertBefore(hint, table);
        }
    });
}

// 검색 결과 개선
function enhanceSearch() {
    // 검색 입력 필드 개선
    const searchInput = document.querySelector('.md-search__input');
    if (searchInput) {
        // 검색 힌트 추가
        searchInput.setAttribute('placeholder', '문서 검색... (Ctrl+K)');
        
        // 키보드 단축키 추가
        document.addEventListener('keydown', function(e) {
            if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
                e.preventDefault();
                searchInput.focus();
            }
        });
    }
    
    // 검색 결과에 카테고리 표시
    const observer = new MutationObserver(function(mutations) {
        mutations.forEach(function(mutation) {
            if (mutation.type === 'childList') {
                const results = document.querySelectorAll('.md-search-result__item');
                results.forEach(addCategoryToResult);
            }
        });
    });
    
    const searchResults = document.querySelector('.md-search-result__list');
    if (searchResults) {
        observer.observe(searchResults, { childList: true, subtree: true });
    }
}

// 검색 결과에 카테고리 추가
function addCategoryToResult(result) {
    const link = result.querySelector('.md-search-result__link');
    if (link && !result.querySelector('.result-category')) {
        const href = link.getAttribute('href');
        const category = getCategoryFromHref(href);
        
        if (category) {
            const categoryBadge = document.createElement('span');
            categoryBadge.className = 'result-category badge info';
            categoryBadge.textContent = category;
            categoryBadge.style.marginLeft = '0.5rem';
            
            const title = result.querySelector('.md-search-result__title');
            if (title) {
                title.appendChild(categoryBadge);
            }
        }
    }
}

// URL에서 카테고리 추출
function getCategoryFromHref(href) {
    const pathMap = {
        'introduction/': '시작하기',
        'user-guide/': '사용자 가이드',
        'api/': 'API',
        'admin/': '관리자',
        'development/': '개발자',
        'security/': '보안',
        'troubleshooting/': '문제해결',
        'migration/': '마이그레이션',
        'reference/': '참조'
    };
    
    for (const [path, category] of Object.entries(pathMap)) {
        if (href.includes(path)) {
            return category;
        }
    }
    return null;
}

// 다크모드 토글 처리
function handleThemeToggle() {
    const themeToggle = document.querySelector('.md-header__button[data-md-component="palette"]');
    
    if (themeToggle) {
        themeToggle.addEventListener('click', function() {
            // 테마 변경 애니메이션
            document.body.style.transition = 'background-color 0.3s ease';
            
            setTimeout(function() {
                document.body.style.transition = '';
            }, 300);
        });
    }
}

// 네비게이션 개선
function enhanceNavigation() {
    // 현재 페이지 하이라이트 개선
    const currentPath = window.location.pathname;
    const navLinks = document.querySelectorAll('.md-nav__link');
    
    navLinks.forEach(function(link) {
        if (link.getAttribute('href') === currentPath) {
            link.style.fontWeight = '600';
            link.style.color = 'var(--md-primary-fg-color)';
        }
    });
    
    // 스크롤 시 목차 하이라이트
    highlightTocOnScroll();
    
    // 페이지 내 앵커 링크 부드러운 스크롤
    enableSmoothScroll();
}

// 목차 하이라이트
function highlightTocOnScroll() {
    const headings = document.querySelectorAll('h1, h2, h3, h4, h5, h6');
    const tocLinks = document.querySelectorAll('.md-nav--secondary .md-nav__link');
    
    if (headings.length === 0 || tocLinks.length === 0) return;
    
    let ticking = false;
    
    function updateToc() {
        const scrollTop = window.pageYOffset;
        let current = null;
        
        headings.forEach(function(heading) {
            const rect = heading.getBoundingClientRect();
            if (rect.top <= 100) {
                current = heading;
            }
        });
        
        tocLinks.forEach(function(link) {
            link.classList.remove('active-toc');
        });
        
        if (current) {
            const currentId = current.id;
            const currentLink = document.querySelector(`.md-nav--secondary .md-nav__link[href="#${currentId}"]`);
            if (currentLink) {
                currentLink.classList.add('active-toc');
            }
        }
        
        ticking = false;
    }
    
    function requestTick() {
        if (!ticking) {
            requestAnimationFrame(updateToc);
            ticking = true;
        }
    }
    
    window.addEventListener('scroll', requestTick);
    
    // 활성 목차 스타일
    const style = document.createElement('style');
    style.textContent = `
        .md-nav--secondary .md-nav__link.active-toc {
            color: var(--md-primary-fg-color) !important;
            font-weight: 600;
            border-left: 2px solid var(--md-primary-fg-color);
            padding-left: 0.6rem;
            margin-left: -0.8rem;
        }
    `;
    document.head.appendChild(style);
}

// 부드러운 스크롤
function enableSmoothScroll() {
    const anchorLinks = document.querySelectorAll('a[href^="#"]');
    
    anchorLinks.forEach(function(link) {
        link.addEventListener('click', function(e) {
            const targetId = this.getAttribute('href').substring(1);
            const targetElement = document.getElementById(targetId);
            
            if (targetElement) {
                e.preventDefault();
                targetElement.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
                
                // URL 업데이트
                history.pushState(null, null, `#${targetId}`);
            }
        });
    });
}

// 피드백 시스템 초기화
function initializeFeedback() {
    // 페이지 하단에 피드백 섹션 추가
    const content = document.querySelector('.md-content__inner');
    if (content) {
        const feedbackSection = createFeedbackSection();
        content.appendChild(feedbackSection);
    }
}

// 피드백 섹션 생성
function createFeedbackSection() {
    const section = document.createElement('div');
    section.className = 'feedback-section';
    section.style.cssText = `
        margin-top: 3rem;
        padding: 2rem;
        border-top: 1px solid var(--md-default-fg-color--lighter);
        text-align: center;
        background: var(--md-default-bg-color);
    `;
    
    section.innerHTML = `
        <h3>이 문서가 도움이 되셨나요?</h3>
        <div class="feedback-buttons" style="margin: 1rem 0;">
            <button class="feedback-btn" data-feedback="yes" style="
                margin: 0 0.5rem;
                padding: 0.5rem 1rem;
                border: 1px solid var(--md-primary-fg-color);
                background: transparent;
                color: var(--md-primary-fg-color);
                border-radius: 0.2rem;
                cursor: pointer;
                transition: all 0.25s;
            ">👍 도움됨</button>
            <button class="feedback-btn" data-feedback="no" style="
                margin: 0 0.5rem;
                padding: 0.5rem 1rem;
                border: 1px solid #f44336;
                background: transparent;
                color: #f44336;
                border-radius: 0.2rem;
                cursor: pointer;
                transition: all 0.25s;
            ">👎 개선필요</button>
        </div>
        <div class="feedback-result" style="display: none; margin-top: 1rem;"></div>
    `;
    
    // 피드백 버튼 이벤트
    const feedbackBtns = section.querySelectorAll('.feedback-btn');
    feedbackBtns.forEach(function(btn) {
        btn.addEventListener('click', function() {
            const feedback = this.getAttribute('data-feedback');
            handleFeedback(feedback, section);
        });
        
        // 호버 효과
        btn.addEventListener('mouseenter', function() {
            if (this.getAttribute('data-feedback') === 'yes') {
                this.style.backgroundColor = 'var(--md-primary-fg-color)';
                this.style.color = 'white';
            } else {
                this.style.backgroundColor = '#f44336';
                this.style.color = 'white';
            }
        });
        
        btn.addEventListener('mouseleave', function() {
            this.style.backgroundColor = 'transparent';
            if (this.getAttribute('data-feedback') === 'yes') {
                this.style.color = 'var(--md-primary-fg-color)';
            } else {
                this.style.color = '#f44336';
            }
        });
    });
    
    return section;
}

// 피드백 처리
function handleFeedback(feedback, section) {
    const buttons = section.querySelector('.feedback-buttons');
    const result = section.querySelector('.feedback-result');
    
    buttons.style.display = 'none';
    result.style.display = 'block';
    
    if (feedback === 'yes') {
        result.innerHTML = `
            <p style="color: green;">피드백을 주셔서 감사합니다! 🎉</p>
            <p><a href="https://github.com/your-org/aicli-web/issues/new?template=documentation.md" target="_blank">문서 개선 제안하기</a></p>
        `;
    } else {
        result.innerHTML = `
            <p style="color: orange;">피드백을 주셔서 감사합니다. 더 나은 문서를 위해 노력하겠습니다.</p>
            <p><a href="https://github.com/your-org/aicli-web/issues/new?template=documentation.md" target="_blank">개선사항 제안하기</a></p>
        `;
    }
    
    // 분석을 위한 이벤트 전송 (Google Analytics 등)
    if (typeof gtag !== 'undefined') {
        gtag('event', 'feedback', {
            'event_category': 'documentation',
            'event_label': window.location.pathname,
            'value': feedback === 'yes' ? 1 : 0
        });
    }
}

// 유틸리티 함수들
const utils = {
    // 디바운스 함수
    debounce: function(func, wait, immediate) {
        let timeout;
        return function executedFunction() {
            const context = this;
            const args = arguments;
            const later = function() {
                timeout = null;
                if (!immediate) func.apply(context, args);
            };
            const callNow = immediate && !timeout;
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
            if (callNow) func.apply(context, args);
        };
    },
    
    // 스로틀 함수
    throttle: function(func, limit) {
        let inThrottle;
        return function() {
            const args = arguments;
            const context = this;
            if (!inThrottle) {
                func.apply(context, args);
                inThrottle = true;
                setTimeout(() => inThrottle = false, limit);
            }
        };
    },
    
    // 로컬 스토리지 헬퍼
    storage: {
        set: function(key, value) {
            try {
                localStorage.setItem(key, JSON.stringify(value));
            } catch (e) {
                console.warn('로컬 스토리지에 저장할 수 없습니다:', e);
            }
        },
        
        get: function(key, defaultValue) {
            try {
                const item = localStorage.getItem(key);
                return item ? JSON.parse(item) : defaultValue;
            } catch (e) {
                console.warn('로컬 스토리지에서 읽을 수 없습니다:', e);
                return defaultValue;
            }
        }
    }
};

// 전역으로 유틸리티 함수 노출
window.docsUtils = utils;