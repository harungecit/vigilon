// Common functions

// Toast notification function
function showToast(message, type = 'error') {
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    toast.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        background: ${type === 'error' ? '#f44336' : type === 'success' ? '#4caf50' : '#ff9800'};
        color: white;
        padding: 16px 24px;
        border-radius: 4px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        z-index: 10000;
        animation: slideIn 0.3s ease-out;
        max-width: 400px;
    `;
    document.body.appendChild(toast);
    
    setTimeout(() => {
        toast.style.animation = 'slideOut 0.3s ease-out';
        setTimeout(() => toast.remove(), 300);
    }, 5000);
}

// Add CSS animations
if (!document.getElementById('toast-styles')) {
    const style = document.createElement('style');
    style.id = 'toast-styles';
    style.textContent = `
        @keyframes slideIn {
            from { transform: translateX(400px); opacity: 0; }
            to { transform: translateX(0); opacity: 1; }
        }
        @keyframes slideOut {
            from { transform: translateX(0); opacity: 1; }
            to { transform: translateX(400px); opacity: 0; }
        }
    `;
    document.head.appendChild(style);
}

// Enhanced fetch wrapper with error handling
async function apiFetch(url, options = {}) {
    try {
        const response = await fetch(url, options);
        
        if (response.status === 401) {
            showToast('Session expired. Please login again.', 'error');
            setTimeout(() => window.location.href = '/login', 2000);
            throw new Error('Unauthorized');
        }
        
        if (response.status === 403) {
            showToast('⚠️ You don\'t have permission to perform this action.', 'error');
            throw new Error('Forbidden');
        }
        
        if (!response.ok) {
            const data = await response.json().catch(() => ({}));
            const message = data.error || `Request failed with status ${response.status}`;
            showToast(message, 'error');
            throw new Error(message);
        }
        
        return response;
    } catch (error) {
        if (error.message !== 'Unauthorized' && error.message !== 'Forbidden') {
            showToast(`Network error: ${error.message}`, 'error');
        }
        throw error;
    }
}

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
    // Close user dropdown if clicking outside
    if (!event.target.closest('.user-dropdown')) {
        const dropdown = document.getElementById('userDropdown');
        if (dropdown) {
            dropdown.classList.remove('show');
        }
    }
}

// Toggle user dropdown menu
function toggleUserMenu(event) {
    event.preventDefault();
    event.stopPropagation();
    const dropdown = document.getElementById('userDropdown');
    if (dropdown) {
        dropdown.classList.toggle('show');
    }
}

// Show change password modal
function showChangePasswordModal() {
    const modal = document.createElement('div');
    modal.className = 'modal';
    modal.style.display = 'block';
    modal.innerHTML = `
        <div class="modal-content">
            <span class="close" onclick="this.closest('.modal').remove()">&times;</span>
            <h3>Change Password</h3>
            <form id="changePasswordForm">
                <div class="form-group">
                    <label for="current_password">Current Password: *</label>
                    <input type="password" id="current_password" name="current_password" required autocomplete="current-password" minlength="4">
                </div>
                <div class="form-group">
                    <label for="new_password">New Password: *</label>
                    <input type="password" id="new_password" name="new_password" required autocomplete="new-password" minlength="4">
                </div>
                <div class="form-group">
                    <label for="confirm_password">Confirm New Password: *</label>
                    <input type="password" id="confirm_password" name="confirm_password" required autocomplete="new-password" minlength="4">
                </div>
                <button type="submit" class="btn">Change Password</button>
            </form>
        </div>
    `;
    document.body.appendChild(modal);

    document.getElementById('changePasswordForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(e.target);
        
        const currentPassword = formData.get('current_password');
        const newPassword = formData.get('new_password');
        const confirmPassword = formData.get('confirm_password');

        if (newPassword !== confirmPassword) {
            showToast('New passwords do not match', 'error');
            return;
        }

        if (newPassword.length < 4) {
            showToast('Password must be at least 4 characters', 'error');
            return;
        }

        try {
            // Get current user ID from session
            const userResponse = await apiFetch('/api/users/me');
            const userData = await userResponse.json();

            const response = await apiFetch(`/api/users/${userData.id}/password`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    current_password: currentPassword,
                    new_password: newPassword
                })
            });

            const result = await response.json();
            showToast(result.message || 'Password changed successfully', 'success');
            modal.remove();
        } catch (error) {
            // Error already shown by apiFetch
        }
    });

    // Close on outside click
    modal.addEventListener('click', (e) => {
        if (e.target === modal) {
            modal.remove();
        }
    });
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
    const confirmed = await Confirm.show({
        title: 'Logout',
        message: 'Are you sure you want to logout?',
        confirmText: 'Logout',
        type: 'warning'
    });
    
    if (!confirmed) return;
    
    try {
        await fetch('/api/auth/logout', { method: 'POST' });
        window.location.href = '/login';
    } catch (error) {
        window.location.href = '/login';
    }
}
