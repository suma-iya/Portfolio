(() => {
  'use strict';

  // ==========================================================================
  // CONFIG
  // ==========================================================================
  // In production the site and API share one origin — on Vercel (vercel.json
  // rewrites /api/v1/contact to the Go function in api/) or on EC2 (Nginx
  // proxies /api/* to the Go backend) — so the base URL is empty and requests
  // go to the same host. Locally the Go server runs on :8080.
  const isLocalDev = ['localhost', '127.0.0.1'].includes(window.location.hostname);
  const CONFIG = {
    API_BASE_URL: window.PORTFOLIO_API_BASE_URL ?? (isLocalDev ? 'http://localhost:8080' : ''),
  };

  // ==========================================================================
  // NAV SCROLL STATE + SCROLL PROGRESS + SCROLL-TOP BUTTON
  // ==========================================================================
  const navbar = document.getElementById('navbar');
  const scrollTopBtn = document.getElementById('scrollTop');
  const scrollProgress = document.getElementById('scrollProgress');

  function onScroll() {
    const scrollY = window.scrollY;
    const docHeight = document.documentElement.scrollHeight - window.innerHeight;
    navbar.classList.toggle('scrolled', scrollY > 50);
    scrollTopBtn.classList.toggle('visible', scrollY > 400);
    scrollProgress.style.width = docHeight > 0 ? `${(scrollY / docHeight) * 100}%` : '0%';
  }
  window.addEventListener('scroll', onScroll, { passive: true });
  onScroll();

  scrollTopBtn.addEventListener('click', () => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  });

  // ==========================================================================
  // HAMBURGER / MOBILE NAV
  // ==========================================================================
  const hamburger = document.getElementById('hamburger');
  const navMobile = document.getElementById('navMobile');

  hamburger.addEventListener('click', () => {
    const isOpen = hamburger.classList.toggle('open');
    navMobile.classList.toggle('open');
    hamburger.setAttribute('aria-expanded', String(isOpen));
  });

  window.closeMobile = function closeMobile() {
    hamburger.classList.remove('open');
    navMobile.classList.remove('open');
    hamburger.setAttribute('aria-expanded', 'false');
  };

  // ==========================================================================
  // ACTIVE NAV LINK ON SCROLL
  // ==========================================================================
  const sections = document.querySelectorAll('main section[id]');
  const navLinks = document.querySelectorAll('.nav-links a');

  const navObserver = new IntersectionObserver((entries) => {
    entries.forEach((entry) => {
      if (entry.isIntersecting) {
        const id = entry.target.getAttribute('id');
        navLinks.forEach((link) => {
          link.classList.toggle('active', link.getAttribute('href') === `#${id}`);
        });
      }
    });
  }, { rootMargin: '-40% 0px -55% 0px' });

  sections.forEach((section) => navObserver.observe(section));

  // ==========================================================================
  // SCROLL REVEAL
  // ==========================================================================
  const reveals = document.querySelectorAll('.reveal');
  const revealObserver = new IntersectionObserver((entries) => {
    entries.forEach((entry, i) => {
      if (entry.isIntersecting) {
        setTimeout(() => entry.target.classList.add('visible'), i * 60);
        revealObserver.unobserve(entry.target);
      }
    });
  }, { threshold: 0.1 });
  reveals.forEach((el) => revealObserver.observe(el));

  // ==========================================================================
  // ANIMATED COUNTERS
  // ==========================================================================
  function animateCounter(el) {
    const target = parseInt(el.dataset.count, 10) || 0;
    const suffix = el.dataset.suffix || '';
    const duration = 1400;
    const startTime = performance.now();

    function tick(now) {
      const progress = Math.min((now - startTime) / duration, 1);
      const eased = 1 - Math.pow(1 - progress, 3);
      el.textContent = Math.round(eased * target) + suffix;
      if (progress < 1) requestAnimationFrame(tick);
    }
    requestAnimationFrame(tick);
  }

  const counters = document.querySelectorAll('[data-count]');
  const counterObserver = new IntersectionObserver((entries) => {
    entries.forEach((entry) => {
      if (entry.isIntersecting) {
        animateCounter(entry.target);
        counterObserver.unobserve(entry.target);
      }
    });
  }, { threshold: 0.5 });
  counters.forEach((el) => counterObserver.observe(el));

  // ==========================================================================
  // TYPING ANIMATION
  // ==========================================================================
  const typingEl = document.getElementById('typingLine');
  const phrases = [
    'Building backend services in Go.',
    'Shipping containerized stacks with Docker Compose.',
    'Publishing research in SoftwareX & IEEE.',
  ];

  function typeLoop() {
    if (!typingEl || window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      if (typingEl) typingEl.textContent = phrases[0];
      return;
    }
    let phraseIndex = 0;
    let charIndex = 0;
    let deleting = false;

    function step() {
      const current = phrases[phraseIndex];
      if (!deleting) {
        charIndex++;
        typingEl.innerHTML = current.slice(0, charIndex) + '<span class="cursor"></span>';
        if (charIndex === current.length) {
          deleting = true;
          setTimeout(step, 1600);
          return;
        }
        setTimeout(step, 45);
      } else {
        charIndex--;
        typingEl.innerHTML = current.slice(0, charIndex) + '<span class="cursor"></span>';
        if (charIndex === 0) {
          deleting = false;
          phraseIndex = (phraseIndex + 1) % phrases.length;
          setTimeout(step, 300);
          return;
        }
        setTimeout(step, 25);
      }
    }
    step();
  }
  typeLoop();

  // ==========================================================================
  // PARALLAX HERO GLOW
  // ==========================================================================
  const heroGlow = document.querySelector('.hero-glow');
  const heroGlow2 = document.querySelector('.hero-glow2');
  if (heroGlow && heroGlow2 && !window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    window.addEventListener('scroll', () => {
      const y = window.scrollY;
      heroGlow.style.transform = `translateY(${y * 0.15}px)`;
      heroGlow2.style.transform = `translateY(${y * -0.1}px)`;
    }, { passive: true });
  }

  // ==========================================================================
  // PROJECT FILTERING
  // ==========================================================================
  const filterBtns = document.querySelectorAll('.filter-btn');
  const projectCards = document.querySelectorAll('.project-card');

  filterBtns.forEach((btn) => {
    btn.addEventListener('click', () => {
      filterBtns.forEach((b) => { b.classList.remove('active'); b.setAttribute('aria-selected', 'false'); });
      btn.classList.add('active');
      btn.setAttribute('aria-selected', 'true');

      const filter = btn.dataset.filter;
      projectCards.forEach((card) => {
        const categories = (card.dataset.category || '').split(' ');
        const show = filter === 'all' || categories.includes(filter);
        card.classList.toggle('hidden', !show);
      });
    });
  });

  // ==========================================================================
  // CONTACT FORM — POST to Go/Gin backend, SMTP-relayed to Gmail
  // ==========================================================================
  const form = document.getElementById('contactForm');
  const formStatus = document.getElementById('formStatus');
  const submitBtn = document.getElementById('submitBtn');

  function setFieldError(fieldId, message) {
    const errorEl = document.getElementById(`${fieldId}Error`);
    const inputEl = document.getElementById(fieldId);
    if (errorEl) errorEl.textContent = message;
    if (inputEl) inputEl.setAttribute('aria-invalid', message ? 'true' : 'false');
  }

  function validateForm(data) {
    let valid = true;
    setFieldError('name', ''); setFieldError('email', ''); setFieldError('message', '');

    if (!data.name || data.name.trim().length < 2) {
      setFieldError('name', 'Please enter your name (min 2 characters).');
      valid = false;
    }
    const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!data.email || !emailPattern.test(data.email)) {
      setFieldError('email', 'Please enter a valid email address.');
      valid = false;
    }
    if (!data.message || data.message.trim().length < 10) {
      setFieldError('message', 'Message should be at least 10 characters.');
      valid = false;
    }
    return valid;
  }

  function showStatus(type, message) {
    formStatus.textContent = message;
    formStatus.className = `form-status show ${type}`;
  }

  if (form) {
    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      const data = {
        name: form.name.value.trim(),
        email: form.email.value.trim(),
        subject: form.subject.value.trim(),
        message: form.message.value.trim(),
      };

      if (!validateForm(data)) {
        showStatus('error', 'Please fix the errors above and try again.');
        return;
      }

      submitBtn.disabled = true;
      submitBtn.querySelector('span').textContent = 'Sending...';
      formStatus.className = 'form-status';

      try {
        const res = await fetch(`${CONFIG.API_BASE_URL}/api/v1/contact`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(data),
        });

        if (!res.ok) {
          const body = await res.json().catch(() => ({}));
          throw new Error(body.message || 'Something went wrong. Please try again.');
        }

        showStatus('success', "Message sent — I'll get back to you soon.");
        form.reset();
      } catch (err) {
        showStatus('error', err.message || 'Failed to send message. Please email me directly.');
      } finally {
        submitBtn.disabled = false;
        submitBtn.querySelector('span').textContent = 'Send Message';
      }
    });
  }
})();
