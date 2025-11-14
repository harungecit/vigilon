// Toast Notification System
const Toast = {
    container: null,
    
    init() {
        if (!this.container) {
            this.container = document.createElement('div');
            this.container.id = 'toast-container';
            document.body.appendChild(this.container);
        }
    },
    
    show(message, type = 'info', title = null, duration = 4000) {
        this.init();
        
        const icons = {
            success: '✓',
            error: '✕',
            warning: '⚠',
            info: 'ℹ'
        };
        
        const titles = {
            success: title || 'Success',
            error: title || 'Error',
            warning: title || 'Warning',
            info: title || 'Information'
        };
        
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.innerHTML = `
            <div class="toast-icon">${icons[type]}</div>
            <div class="toast-content">
                <div class="toast-title">${titles[type]}</div>
                <div class="toast-message">${message}</div>
            </div>
            <button class="toast-close" onclick="Toast.close(this.parentElement)">&times;</button>
        `;
        
        this.container.appendChild(toast);
        
        // Auto remove after duration
        if (duration > 0) {
            setTimeout(() => {
                this.close(toast);
            }, duration);
        }
        
        return toast;
    },
    
    close(toast) {
        if (toast && toast.parentElement) {
            toast.classList.add('removing');
            setTimeout(() => {
                if (toast.parentElement) {
                    toast.parentElement.removeChild(toast);
                }
            }, 300);
        }
    },
    
    success(message, title = null, duration = 4000) {
        return this.show(message, 'success', title, duration);
    },
    
    error(message, title = null, duration = 5000) {
        return this.show(message, 'error', title, duration);
    },
    
    warning(message, title = null, duration = 4500) {
        return this.show(message, 'warning', title, duration);
    },
    
    info(message, title = null, duration = 4000) {
        return this.show(message, 'info', title, duration);
    }
};

// Confirmation Modal System
const Confirm = {
    modal: null,
    resolveCallback: null,
    
    init() {
        if (!this.modal) {
            this.modal = document.createElement('div');
            this.modal.id = 'confirm-modal';
            this.modal.innerHTML = `
                <div class="modal-content">
                    <div class="modal-header">
                        <div class="modal-icon">
                            <span class="modal-icon-text"></span>
                        </div>
                        <h3 class="modal-title"></h3>
                    </div>
                    <div class="modal-body">
                        <p class="modal-message"></p>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-cancel" onclick="Confirm.cancel()">Cancel</button>
                        <button class="btn btn-confirm" onclick="Confirm.confirm()">Confirm</button>
                    </div>
                </div>
            `;
            document.body.appendChild(this.modal);
        }
    },
    
    show(options = {}) {
        this.init();
        
        const {
            title = 'Confirm Action',
            message = 'Are you sure you want to proceed?',
            confirmText = 'Confirm',
            cancelText = 'Cancel',
            type = 'warning', // warning, danger, info
            icon = type === 'danger' ? '⚠' : (type === 'info' ? 'ℹ' : '?')
        } = options;
        
        // Update modal content
        this.modal.querySelector('.modal-title').textContent = title;
        this.modal.querySelector('.modal-message').textContent = message;
        this.modal.querySelector('.modal-icon-text').textContent = icon;
        this.modal.querySelector('.btn-confirm').textContent = confirmText;
        this.modal.querySelector('.btn-cancel').textContent = cancelText;
        
        // Update icon style
        const iconEl = this.modal.querySelector('.modal-icon');
        iconEl.className = `modal-icon ${type}`;
        
        // Update confirm button style
        const confirmBtn = this.modal.querySelector('.btn-confirm');
        confirmBtn.className = `btn btn-confirm ${type === 'danger' ? 'danger' : ''}`;
        
        // Show modal
        this.modal.style.display = 'block';
        
        // Return promise
        return new Promise((resolve) => {
            this.resolveCallback = resolve;
        });
    },
    
    confirm() {
        if (this.resolveCallback) {
            this.resolveCallback(true);
            this.resolveCallback = null;
        }
        this.hide();
    },
    
    cancel() {
        if (this.resolveCallback) {
            this.resolveCallback(false);
            this.resolveCallback = null;
        }
        this.hide();
    },
    
    hide() {
        if (this.modal) {
            this.modal.style.display = 'none';
        }
    }
};

// Close modal on outside click
document.addEventListener('click', (e) => {
    if (e.target.id === 'confirm-modal') {
        Confirm.cancel();
    }
});

// Close modal on Escape key
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && Confirm.modal && Confirm.modal.style.display === 'block') {
        Confirm.cancel();
    }
});

// Backward compatibility - replace native alert
window.showToast = Toast.show.bind(Toast);
window.showConfirm = Confirm.show.bind(Confirm);
