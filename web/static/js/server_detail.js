function showAddServiceModal() {
    document.getElementById('addServiceModal').style.display = 'block';
}

function toggleEdit() {
    const infoTable = document.getElementById('serverInfo');
    const editForm = document.getElementById('editServerForm');

    if (editForm.style.display === 'none') {
        infoTable.style.display = 'none';
        editForm.style.display = 'block';
    } else {
        infoTable.style.display = 'table';
        editForm.style.display = 'none';
    }
}

function toggleAgentScript() {
    const section = document.getElementById('agentScriptSection');

    if (section.style.display === 'none') {
        section.style.display = 'block';
        // Load agent script immediately (it's fast now - just a one-line command)
        loadAgentScript();
        // Save state to localStorage
        localStorage.setItem('agentScriptVisible_' + serverData.id, 'true');
    } else {
        section.style.display = 'none';
        // Save state to localStorage
        localStorage.setItem('agentScriptVisible_' + serverData.id, 'false');
    }
}

async function loadAgentScript() {
    try {
        // Use the new one-line installer
        const installCommand = `curl -fsSL ${window.location.origin}/install.sh?token=${serverData.token} | sudo bash`;
        document.getElementById('agentInstallScript').textContent = installCommand;
    } catch (error) {
        document.getElementById('agentInstallScript').textContent = 'Error: ' + error.message;
    }
}

function copyAgentScript(event) {
    const script = document.getElementById('agentInstallScript').textContent;
    const btn = event.target.closest('button');
    if (!btn) return;
    
    const originalText = btn.textContent;
    const originalBg = btn.style.background || '#3498db';

    navigator.clipboard.writeText(script).then(() => {
        btn.textContent = '✓ Copied!';
        btn.style.background = '#27ae60';
        setTimeout(() => {
            btn.textContent = originalText;
            btn.style.background = originalBg;
        }, 2000);
    }).catch(err => {
        btn.textContent = '✗ Failed';
        btn.style.background = '#e74c3c';
        setTimeout(() => {
            btn.textContent = originalText;
            btn.style.background = originalBg;
        }, 2000);
        console.error('Failed to copy:', err);
    });
}

function copyCommand(event, command) {
    const btn = event.target.closest('button');
    if (!btn) return;
    
    const originalText = btn.textContent;
    const originalBg = btn.style.background || '#3498db';

    navigator.clipboard.writeText(command).then(() => {
        btn.textContent = '✓ Copied!';
        btn.style.background = '#27ae60';
        setTimeout(() => {
            btn.textContent = originalText;
            btn.style.background = originalBg;
        }, 2000);
    }).catch(err => {
        btn.textContent = '✗ Failed';
        btn.style.background = '#e74c3c';
        setTimeout(() => {
            btn.textContent = originalText;
            btn.style.background = originalBg;
        }, 2000);
        console.error('Failed to copy:', err);
    });
}

async function toggleServerStatus(serverId, currentStatus) {
    const newStatus = !currentStatus;
    const action = newStatus ? 'enable' : 'disable';

    if (!confirm(`Are you sure you want to ${action} this server?`)) {
        return;
    }

    try {
        // Get current server data
        const getResponse = await fetch(`/api/servers/${serverId}`);
        if (!getResponse.ok) {
            throw new Error('Failed to fetch server data');
        }

        const server = await getResponse.json();
        server.enabled = newStatus;

        // Update server
        const response = await fetch(`/api/servers/${serverId}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(server),
        });

        if (response.ok) {
            alert(`Server ${action}d successfully!`);
            window.location.reload();
        } else {
            const error = await response.json();
            alert('Failed to update server: ' + (error.error || 'Unknown error'));
        }
    } catch (error) {
        alert('Failed to update server: ' + error.message);
    }
}

async function toggleServiceStatus(serviceId, currentStatus) {
    const newStatus = !currentStatus;
    const action = newStatus ? 'enable' : 'disable';

    if (!confirm(`Are you sure you want to ${action} this service?`)) {
        return;
    }

    try {
        // Get current service data
        const getResponse = await fetch(`/api/services/${serviceId}`);
        if (!getResponse.ok) {
            // Service endpoint doesn't exist, we need to create one
            // For now, just show error
            throw new Error('Service API endpoint not available');
        }

        const service = await getResponse.json();
        service.enabled = newStatus;

        // Update service
        const response = await fetch(`/api/services/${serviceId}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(service),
        });

        if (response.ok) {
            alert(`Service ${action}d successfully!`);
            window.location.reload();
        } else {
            const error = await response.json();
            alert('Failed to update service: ' + (error.error || 'Unknown error'));
        }
    } catch (error) {
        alert('Failed to update service: ' + error.message);
    }
}

// Edit server form submission
document.getElementById('editServerForm').addEventListener('submit', async (e) => {
    e.preventDefault();

    const formData = new FormData(e.target);

    try {
        // Get current server data
        const getResponse = await fetch(`/api/servers/${serverData.id}`);
        if (!getResponse.ok) {
            throw new Error('Failed to fetch server data');
        }

        const server = await getResponse.json();

        // Update with form data
        server.name = formData.get('name');
        server.hostname = formData.get('hostname');
        server.ip_address = formData.get('ip_address');
        server.port = parseInt(formData.get('port'));
        server.check_interval = parseInt(formData.get('check_interval'));
        server.notify_telegram = formData.get('notify_telegram') === 'on';

        // Update server
        const response = await fetch(`/api/servers/${serverData.id}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(server),
        });

        if (response.ok) {
            alert('Server updated successfully!');
            window.location.reload();
        } else {
            const error = await response.json();
            alert('Failed to update server: ' + (error.error || 'Unknown error'));
        }
    } catch (error) {
        alert('Failed to update server: ' + error.message);
    }
});

document.getElementById('addServiceForm').addEventListener('submit', async (e) => {
    e.preventDefault();

    const formData = new FormData(e.target);
    const data = {
        server_id: parseInt(formData.get('server_id')),
        name: formData.get('name'),
        display_name: formData.get('display_name'),
        description: formData.get('description') || '',
        enabled: formData.get('enabled') === 'on'
    };

    try {
        const response = await fetch('/api/services', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(data),
        });

        if (response.ok) {
            alert('Service added successfully!');
            window.location.reload();
        } else {
            const error = await response.json();
            alert('Failed to add service: ' + (error.error || 'Unknown error'));
        }
    } catch (error) {
        alert('Failed to add service: ' + error.message);
    }
});

function deleteService(serviceId) {
    if (!confirm('Are you sure you want to delete this service?')) {
        return;
    }

    fetch(`/api/services/${serviceId}`, {
        method: 'DELETE'
    })
    .then(response => {
        if (response.ok) {
            location.reload();
        } else {
            alert('Failed to delete service');
        }
    })
    .catch(error => {
        console.error('Error deleting service:', error);
        alert('Failed to delete service');
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

async function viewServiceHistory(id) {
    // Show modal
    const modal = document.getElementById('serviceHistoryModal');
    const content = document.getElementById('serviceHistoryContent');
    modal.style.display = 'block';
    content.innerHTML = '<div class="loading">Loading history...</div>';
    
    try {
        const response = await fetch(`/api/services/${id}/checks?limit=20`);
        if (response.ok) {
            const checks = await response.json();
            
            if (checks.length === 0) {
                content.innerHTML = '<p>No check history available yet.</p>';
                return;
            }
            
            let html = '<table class="table"><thead><tr><th>Date & Time</th><th>Status</th><th>PID</th><th>CPU</th><th>Memory</th><th>Error</th></tr></thead><tbody>';
            
            checks.forEach(check => {
                const date = new Date(check.checked_at).toLocaleString();
                const statusClass = check.status === 'running' ? 'badge-success' : 
                                  check.status === 'stopped' ? 'badge-danger' : 'badge-warning';
                html += `<tr>
                    <td>${date}</td>
                    <td><span class="badge ${statusClass}">${check.status}</span></td>
                    <td>${check.pid || '-'}</td>
                    <td>${check.cpu ? check.cpu.toFixed(1) + '%' : '-'}</td>
                    <td>${check.memory ? check.memory.toFixed(1) + ' MB' : '-'}</td>
                    <td style="max-width: 300px; overflow: hidden; text-overflow: ellipsis;" title="${check.error_message || ''}">${check.error_message || '-'}</td>
                </tr>`;
            });
            
            html += '</tbody></table>';
            content.innerHTML = html;
        } else {
            content.innerHTML = '<p class="error">Failed to load service history</p>';
        }
    } catch (error) {
        content.innerHTML = '<p class="error">Failed to load service history: ' + error.message + '</p>';
    }
}

// Initialize page state on load
document.addEventListener('DOMContentLoaded', function() {
    // Restore agent script visibility state
    const agentScriptSection = document.getElementById('agentScriptSection');
    if (agentScriptSection) {
        const isVisible = localStorage.getItem('agentScriptVisible_' + serverData.id);
        if (isVisible === 'true') {
            agentScriptSection.style.display = 'block';
            loadAgentScript();
        }
    }
});
