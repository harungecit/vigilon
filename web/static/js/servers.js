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
        const response = await fetch('/api/servers', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(data),
        });

        if (response.ok) {
            const server = await response.json();

            // Close modal and redirect to server details page
            closeModal();

            if (mode === 'push') {
                alert('Server added successfully! Redirecting to server details to get the installation command...');
                window.location.href = `/server/${server.id}`;
            } else {
                alert('Server added successfully!');
                window.location.reload();
            }
        } else {
            const error = await response.json();
            alert('Failed to add server: ' + (error.error || 'Unknown error'));
        }
    } catch (error) {
        alert('Failed to add server: ' + error.message);
    }
});

async function disconnectServer(id) {
    if (!confirm('Are you sure you want to disconnect this server? Agent will stop being monitored.')) {
        return;
    }

    try {
        const response = await fetch(`/api/servers/${id}/disconnect`, {
            method: 'POST',
        });

        if (response.ok) {
            alert('Server disconnected successfully!');
            window.location.reload();
        } else {
            const error = await response.json();
            alert('Failed to disconnect server: ' + (error.error || 'Unknown error'));
        }
    } catch (error) {
        alert('Failed to disconnect server: ' + error.message);
    }
}

function deleteServer(serverId) {
    if (!confirm('Are you sure you want to delete this server?')) {
        return;
    }

    fetch(`/api/servers/${serverId}`, {
        method: 'DELETE'
    })
    .then(response => {
        if (response.ok) {
            location.reload();
        } else {
            alert('Failed to delete server');
        }
    })
    .catch(error => {
        console.error('Error deleting server:', error);
        alert('Failed to delete server');
    });
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
