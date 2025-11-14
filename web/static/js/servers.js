let currentServerURL = window.location.origin;

function showAddServerModal() {
    document.getElementById('addServerModal').style.display = 'block';
}

function updateFormFields() {
    const mode = document.getElementById('monitoringMode').value;
    const pushFields = document.getElementById('pushModeFields');
    const pullFields = document.getElementById('pullModeFields');
    const modeInfo = document.getElementById('modeInfo');
    const modeInfoText = document.getElementById('modeInfoText');

    // Hide all conditional fields first
    pushFields.classList.add('hidden');
    pullFields.classList.add('hidden');
    modeInfo.style.display = 'none';

    // Show relevant fields based on mode
    if (mode === 'push') {
        pushFields.classList.remove('hidden');
        modeInfo.style.display = 'block';
        modeInfoText.innerHTML = '<strong>Push Mode:</strong> Lightweight agent runs on target server and reports status to Vigilon. No SSH required. Best for servers behind NAT or firewalls.';
    } else if (mode === 'pull') {
        pullFields.classList.remove('hidden');
        modeInfo.style.display = 'block';
        modeInfoText.innerHTML = '<strong>Pull Mode:</strong> Vigilon connects to target server via SSH to check service status. Requires SSH access and credentials.';
    } else if (mode === 'hybrid') {
        pullFields.classList.remove('hidden');
        modeInfo.style.display = 'block';
        modeInfoText.innerHTML = '<strong>Hybrid Mode:</strong> Combines SSH access with local scripts for flexible monitoring. Requires SSH access.';
    }
}

function toggleJumpHost() {
    const useJumpHost = document.getElementById('useJumpHost').checked;
    const jumpHostFields = document.getElementById('jumpHostFields');

    if (useJumpHost) {
        jumpHostFields.classList.remove('hidden');
    } else {
        jumpHostFields.classList.add('hidden');
    }
}

function generateToken() {
    // Generate a random token
    const token = Array.from(crypto.getRandomValues(new Uint8Array(32)))
        .map(b => b.toString(16).padStart(2, '0'))
        .join('');
    document.getElementById('agentToken').value = token;
}


document.getElementById('addServerForm').addEventListener('submit', async (e) => {
    e.preventDefault();

    const formData = new FormData(e.target);
    const mode = formData.get('monitoring_mode');

    // Auto-generate token if push mode and no token provided
    let token = formData.get('agent_token');
    if (mode === 'push' && !token) {
        token = Array.from(crypto.getRandomValues(new Uint8Array(32)))
            .map(b => b.toString(16).padStart(2, '0'))
            .join('');
    }

    const data = {
        name: formData.get('name'),
        hostname: formData.get('hostname') || formData.get('name'),
        ip_address: formData.get('ip_address'),
        port: parseInt(formData.get('port')) || 22,
        os: formData.get('os'),
        monitoring_mode: mode,
        ssh_user: formData.get('ssh_user') || '',
        ssh_key_path: formData.get('ssh_key_path') || '',
        ssh_jump_host: formData.get('ssh_jump_host') || '',
        ssh_jump_user: formData.get('ssh_jump_user') || '',
        ssh_jump_key_path: formData.get('ssh_jump_key_path') || '',
        agent_token: token || '',
        check_interval: parseInt(formData.get('check_interval')) || 0,
        enabled: formData.get('enabled') === 'on',
        notify_telegram: formData.get('notify_telegram') === 'on'
    };

    try {
        const response = await apiFetch('/api/servers', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(data),
        });

        const server = await response.json();

        // Close modal and redirect to server details page
        closeModal();

        if (mode === 'push') {
            showToast('Server added successfully! Redirecting...', 'success');
            setTimeout(() => window.location.href = `/server/${server.id}`, 1000);
        } else {
            showToast('Server added successfully!', 'success');
            setTimeout(() => window.location.reload(), 1000);
        }
    } catch (error) {
        // Error already handled by apiFetch
    }
});

async function disconnectServer(id) {
    const confirmed = await Confirm.show({
        title: 'Disconnect Server',
        message: 'Are you sure you want to disconnect this server? Agent will stop being monitored.',
        confirmText: 'Disconnect',
        type: 'warning'
    });
    
    if (!confirmed) return;

    try {
        await apiFetch(`/api/servers/${id}/disconnect`, {
            method: 'POST',
        });

        Toast.success('Server disconnected successfully!');
        setTimeout(() => window.location.reload(), 1000);
    } catch (error) {
        // Error already handled by apiFetch
    }
}

async function deleteServer(serverId) {
    const confirmed = await Confirm.show({
        title: 'Delete Server',
        message: 'Are you sure you want to delete this server? This action cannot be undone.',
        confirmText: 'Delete',
        type: 'danger'
    });
    
    if (!confirmed) return;

    try {
        await apiFetch(`/api/servers/${serverId}`, {
            method: 'DELETE'
        });
        
        Toast.success('Server deleted successfully!');
        setTimeout(() => location.reload(), 1000);
    } catch (error) {
        // Error already handled by apiFetch
    }
}

async function logout() {
    try {
        const response = await fetch('/api/auth/logout', {
            method: 'POST'
        });
        
        if (response.ok) {
            window.location.href = '/login';
        }
    } catch (error) {
        console.error('Logout error:', error);
    }
}
