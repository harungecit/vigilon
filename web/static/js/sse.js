// SSE (Server-Sent Events) Client

class SSEClient {
    constructor(endpoint) {
        this.endpoint = endpoint;
        this.eventSource = null;
        this.reconnectDelay = 1000;
        this.maxReconnectDelay = 30000;
        this.currentReconnectDelay = this.reconnectDelay;
        this.listeners = {};
        this.connected = false;
    }

    connect() {
        if (this.eventSource) {
            return;
        }

        this.eventSource = new EventSource(this.endpoint);

        this.eventSource.onopen = () => {
            this.connected = true;
            this.currentReconnectDelay = this.reconnectDelay;
            this.trigger('connected', {});
        };

        this.eventSource.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                this.trigger(data.type, data.data);
            } catch (error) {
                console.error('[SSE] Error parsing message:', error);
            }
        };

        this.eventSource.onerror = (error) => {
            this.connected = false;
            this.eventSource.close();
            this.eventSource = null;
            this.trigger('disconnected', {});
            this.scheduleReconnect();
        };
    }

    scheduleReconnect() {
        setTimeout(() => {
            this.connect();
            this.currentReconnectDelay = Math.min(
                this.currentReconnectDelay * 2,
                this.maxReconnectDelay
            );
        }, this.currentReconnectDelay);
    }

    on(eventType, callback) {
        if (!this.listeners[eventType]) {
            this.listeners[eventType] = [];
        }
        this.listeners[eventType].push(callback);
    }

    off(eventType, callback) {
        if (!this.listeners[eventType]) {
            return;
        }
        this.listeners[eventType] = this.listeners[eventType].filter(
            cb => cb !== callback
        );
    }

    trigger(eventType, data) {
        if (!this.listeners[eventType]) {
            return;
        }
        this.listeners[eventType].forEach(callback => {
            try {
                callback(data);
            } catch (error) {
                // Silently handle errors in callbacks
            }
        });
    }

    disconnect() {
        if (this.eventSource) {
            this.eventSource.close();
            this.eventSource = null;
            this.connected = false;
        }
    }

    isConnected() {
        return this.connected;
    }
}

// Export for use in other scripts
window.SSEClient = SSEClient;
