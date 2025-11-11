// Common functions

function closeModal() {
    const modals = document.querySelectorAll('.modal');
    modals.forEach(modal => {
        modal.style.display = 'none';
    });
}

// Close modal when clicking outside
window.onclick = function(event) {
    if (event.target.classList.contains('modal')) {
        closeModal();
    }
}

// Auto-refresh page every 30 seconds (disabled on users page)
let autoRefresh = !window.location.pathname.includes('/users');
setInterval(() => {
    if (autoRefresh && !document.querySelector('.modal[style*="display: block"]')) {
        window.location.reload();
    }
}, 30000);

// Prevent refresh during form submission
document.addEventListener('submit', () => {
    autoRefresh = false;
});

async function logout() {
    if (!confirm('Are you sure you want to logout?')) {
        return;
    }
    
    try {
        await fetch('/api/auth/logout', { method: 'POST' });
        window.location.href = '/login';
    } catch (error) {
        window.location.href = '/login';
    }
}
