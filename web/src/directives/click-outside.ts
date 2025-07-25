import type { Directive, DirectiveBinding } from 'vue'

interface ClickOutsideElement extends HTMLElement {
  _clickOutsideHandler?: (event: Event) => void;
}

const clickOutside: Directive = {
  mounted(el: ClickOutsideElement, binding: DirectiveBinding) {
    const handler = (event: Event) => {
      // 클릭된 요소가 현재 요소 내부에 있는지 확인
      if (el && !el.contains(event.target as Node)) {
        // 바인딩된 함수 실행
        if (typeof binding.value === 'function') {
          binding.value(event)
        }
      }
    }

    // 핸들러를 요소에 저장
    el._clickOutsideHandler = handler

    // 문서에 이벤트 리스너 추가 (캡처 단계에서)
    document.addEventListener('click', handler, true)
    document.addEventListener('touchstart', handler, true)
  },

  beforeUnmount(el: ClickOutsideElement) {
    // 저장된 핸들러가 있으면 제거
    if (el._clickOutsideHandler) {
      document.removeEventListener('click', el._clickOutsideHandler, true)
      document.removeEventListener('touchstart', el._clickOutsideHandler, true)
      delete el._clickOutsideHandler
    }
  },
}

export default clickOutside