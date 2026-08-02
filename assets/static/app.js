/* ═══════════════════
   NukeCoke's Blog — JS
   ═══════════════════ */

document.addEventListener('DOMContentLoaded', () => {

    // Reply toggle
    document.querySelectorAll('.toggle-reply').forEach(btn => {
        btn.addEventListener('click', () => {
            const box = btn.closest('.comment').querySelector('.comment-reply-box');
            if (!box) return;
            const show = !box.classList.contains('show');
            box.classList.toggle('show');
            btn.textContent = show ? '取消' : '回复';
            if (show) box.querySelector('textarea')?.focus();
        });
    });

    // Fade-in on scroll
    const obs = new IntersectionObserver((entries) => {
        entries.forEach(e => {
            if (e.isIntersecting) {
                e.target.style.opacity = '1';
                e.target.style.transform = 'translateY(0)';
            }
        });
    }, { threshold: .08 });

    document.querySelectorAll('.fade-up').forEach(el => {
        el.style.opacity = '0';
        el.style.transform = 'translateY(12px)';
        el.style.transition = 'opacity .45s ease, transform .45s ease';
        obs.observe(el);
    });
});
