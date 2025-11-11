let currentOffset = 0;
const limit = 10;
let loading = false;
let hasMore = true;

async function acknowledgeAlert(id) {
    try {
        const response = await fetch(`/api/alerts/${id}/acknowledge`, {
            method: 'POST',
        });

        if (response.ok) {
            // Update UI without reload
            const alertCard = document.querySelector(`[data-alert-id="${id}"]`);
            if (alertCard) {
                alertCard.classList.add('acknowledged');
                const footer = alertCard.querySelector('.alert-footer');
                if (footer) {
                    footer.innerHTML = '<span class="acknowledged-badge">Acknowledged just now</span>';
                }
            }
        } else {
            const error = await response.json();
            alert('Failed to acknowledge alert: ' + (error.error || 'Unknown error'));
        }
    } catch (error) {
        alert('Failed to acknowledge alert: ' + error.message);
    }
}

async function archiveAllAlerts() {
    if (!confirm('Are you sure you want to archive all alerts? This will remove them from the list but keep them in the database.')) {
        return;
    }

    try {
        const response = await fetch('/api/alerts/archive-all', {
            method: 'POST',
        });

        if (response.ok) {
            alert('All alerts archived successfully!');
            window.location.reload();
        } else {
            const error = await response.json();
            alert('Failed to archive alerts: ' + (error.error || 'Unknown error'));
        }
    } catch (error) {
        alert('Failed to archive alerts: ' + error.message);
    }
}

async function loadMoreAlerts() {
    if (loading || !hasMore) return;

    loading = true;
    const container = document.querySelector('.alerts-list');
    const loadMoreBtn = document.getElementById('loadMoreBtn');
    const loadingDiv = document.getElementById('loading-indicator');

    if (!container) {
        console.error('Alerts container not found');
        loading = false;
        return;
    }

    if (loadMoreBtn) {
        loadMoreBtn.disabled = true;
        loadMoreBtn.textContent = 'Loading...';
    }

    if (loadingDiv) {
        loadingDiv.style.display = 'block';
    }

    try {
        const response = await fetch(`/api/alerts?limit=${limit}&offset=${currentOffset}`);
        if (response.ok) {
            const alerts = await response.json();

            if (!alerts || alerts.length === 0) {
                hasMore = false;
                if (loadMoreBtn) {
                    loadMoreBtn.style.display = 'none';
                }
                if (loadingDiv) {
                    loadingDiv.textContent = 'No more alerts';
                    loadingDiv.style.display = 'block';
                }
                return;
            }

            alerts.forEach(alert => {
                const alertCard = createAlertCard(alert);
                container.appendChild(alertCard);
            });

            currentOffset += alerts.length;

            if (alerts.length < limit) {
                hasMore = false;
                if (loadMoreBtn) {
                    loadMoreBtn.style.display = 'none';
                }
            }
        } else {
            throw new Error('Failed to fetch alerts');
        }
    } catch (error) {
        console.error('Failed to load alerts:', error);
        alert('Failed to load more alerts: ' + error.message);
    } finally {
        loading = false;
        if (loadMoreBtn) {
            loadMoreBtn.disabled = false;
            loadMoreBtn.textContent = 'Load More';
        }
        if (loadingDiv && hasMore) {
            loadingDiv.style.display = 'none';
        }
    }
}

function createAlertCard(alert) {
    const card = document.createElement('div');
    card.className = `alert-card ${alert.acknowledged ? 'acknowledged' : ''}`;
    card.setAttribute('data-alert-id', alert.id);

    const statusClass = getStatusClass(alert.status);
    const date = new Date(alert.created_at);
    const formattedDate = date.toLocaleString();

    card.innerHTML = `
        <div class="alert-header">
            <span class="alert-id">#${alert.id}</span>
            <span class="alert-status status-${statusClass}">${alert.status}</span>
            <span class="alert-time">${formattedDate}</span>
        </div>
        <div class="alert-body">
            <p>${escapeHtml(alert.message)}</p>
        </div>
        <div class="alert-footer">
            <span class="alert-via">Sent via: ${alert.sent_via}</span>
            ${alert.acknowledged
                ? `<span class="acknowledged-badge">Acknowledged ${new Date(alert.acknowledged_at).toLocaleString()}</span>`
                : `<button class="btn btn-sm" onclick="acknowledgeAlert(${alert.id})">Acknowledge</button>`
            }
        </div>
    `;

    return card;
}

function getStatusClass(status) {
    const statusMap = {
        'running': 'running',
        'stopped': 'stopped',
        'failed': 'failed',
        'degraded': 'degraded',
        'unknown': 'unknown'
    };
    return statusMap[status] || 'unknown';
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
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
