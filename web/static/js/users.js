let roles = [];
let currentRoleId = null;
let currentRoleName = null;
let allPermissions = [];
let previousModal = null; // Track previous modal for navigation
let currentUser = null; // Current logged-in user

async function loadRoles() {
    try {
        const response = await fetch('/api/roles');
        if (response.ok) {
            roles = await response.json();
            updateRoleSelect();
        }
    } catch (error) {
        console.error('Failed to load roles:', error);
    }
}

function updateRoleSelect() {
    const select = document.getElementById('roleId');
    select.innerHTML = '<option value="">Select Role...</option>';
    
    roles.forEach(role => {
        const option = document.createElement('option');
        option.value = role.id;
        option.textContent = role.display_name;
        select.appendChild(option);
    });
}

async function loadCurrentUser() {
    try {
        const response = await fetch('/api/users/me');
        if (response.ok) {
            currentUser = await response.json();
        }
    } catch (error) {
        console.error('Failed to load current user:', error);
    }
}

function hasPermission(permissionName) {
    if (!currentUser) {
        console.warn('hasPermission called but currentUser not loaded');
        return false;
    }
    if (currentUser.role && currentUser.role.is_super_admin) return true;
    return currentUser.role && currentUser.role.permissions && currentUser.role.permissions.some(p => p.name === permissionName);
}

function canViewRoles() {
    if (!currentUser || !currentUser.role) return false;
    // Super Admin can view all roles
    if (currentUser.role.is_super_admin) return true;
    // Admin can view roles
    if (currentUser.role.name === 'Admin') return true;
    // Others cannot view roles
    return false;
}

function canEditRole(role) {
    if (!currentUser || !currentUser.role) return false;
    
    // Super Admin can edit everything EXCEPT Super Admin role itself
    if (currentUser.role.is_super_admin) {
        return !role.is_super_admin;
    }
    
    // Admin can edit User and Custom roles only (NOT Super Admin or Admin)
    if (currentUser.role.name === 'Admin') {
        if (role.is_super_admin) return false; // Cannot edit Super Admin
        if (role.name === 'Admin') return false; // Cannot edit Admin role
        return true; // Can edit User and Custom roles
    }
    
    // User and Custom roles cannot edit any roles
    return false;
}

function canDeleteRole(role) {
    if (!currentUser || !currentUser.role) return false;
    
    // Cannot delete system roles (Super Admin, Admin, User)
    if (role.is_system || role.is_super_admin) return false;
    if (role.name === 'Admin' || role.name === 'User') return false;
    
    // Super Admin can delete custom roles
    if (currentUser.role.is_super_admin) return true;
    
    // Admin can delete custom roles
    if (currentUser.role.name === 'Admin') return true;
    
    // Others cannot delete roles
    return false;
}

function canCreateRole() {
    if (!currentUser || !currentUser.role) return false;
    // Only Super Admin and Admin can create new roles
    return currentUser.role.is_super_admin || currentUser.role.name === 'Admin';
}

async function loadUsers() {
    const container = document.getElementById('users-container');
    
    // Load current user first if not loaded
    if (!currentUser) {
        await loadCurrentUser();
    }
    
    try {
        const response = await fetch('/api/users');
        if (!response.ok) {
            throw new Error('Failed to load users');
        }
        
        const users = await response.json();
        
        if (users.length === 0) {
            container.innerHTML = '<div class="empty-state"><p>No users found.</p></div>';
            return;
        }
        
        let html = '<table class="table"><thead><tr><th>Username</th><th>Email</th><th>Role</th><th>Status</th><th>Last Login</th><th>Actions</th></tr></thead><tbody>';
        
        users.forEach(user => {
            const lastLogin = user.last_login_at ? new Date(user.last_login_at).toLocaleString() : 'Never';
            const roleClass = user.role.is_super_admin ? 'badge-danger' : (user.role.name === 'admin' ? 'badge-warning' : 'badge-info');
            
            // Check permissions for actions
            const canEdit = hasPermission('users.edit');
            const canDelete = hasPermission('users.delete');
            
            // Don't allow non-super-admins to edit super-admins
            const isSuperAdmin = currentUser && currentUser.role && currentUser.role.is_super_admin;
            const canEditThisUser = canEdit && (!user.role.is_super_admin || isSuperAdmin);
            const canDeleteThisUser = canDelete && !user.role.is_super_admin;
            
            html += `
                <tr>
                    <td><strong>${user.username}</strong></td>
                    <td>${user.email}</td>
                    <td>
                        <span class="badge ${roleClass}" style="cursor: pointer;" onclick="showRolePermissions(${user.role.id}, '${escapeHtml(user.role.display_name)}')" title="Click to view permissions">
                            ${user.role.display_name}
                        </span>
                    </td>
                    <td><span class="badge ${user.enabled ? 'badge-success' : 'badge-secondary'}">${user.enabled ? 'Enabled' : 'Disabled'}</span></td>
                    <td>${lastLogin}</td>
                    <td>
                        ${canEditThisUser ? `<button class="btn btn-sm" onclick="editUser(${user.id})">Edit</button>` : ''}
                        ${canEditThisUser ? `<button class="btn btn-sm" onclick="changePassword(${user.id})">Password</button>` : ''}
                        ${canDeleteThisUser ? `<button class="btn btn-sm btn-danger" onclick="deleteUser(${user.id}, '${escapeHtml(user.username)}')">Delete</button>` : ''}
                    </td>
                </tr>
            `;
        });
        
        html += '</tbody></table>';
        container.innerHTML = html;
        
    } catch (error) {
        container.innerHTML = `<div class="error">Failed to load users: ${error.message}</div>`;
    }
}

function showAddUserModal() {
    document.getElementById('modalTitle').textContent = 'Add New User';
    document.getElementById('userForm').reset();
    document.getElementById('userId').value = '';
    document.getElementById('passwordGroup').style.display = 'block';
    document.getElementById('password').required = true;
    document.getElementById('enabled').checked = true;
    document.getElementById('userModal').style.display = 'block';
}

async function editUser(id) {
    try {
        const response = await fetch(`/api/users/${id}`);
        if (!response.ok) {
            throw new Error('Failed to load user');
        }
        
        const user = await response.json();
        
        document.getElementById('modalTitle').textContent = 'Edit User';
        document.getElementById('userId').value = user.id;
        document.getElementById('username').value = user.username;
        document.getElementById('email').value = user.email;
        document.getElementById('roleId').value = user.role_id;
        document.getElementById('enabled').checked = user.enabled;
        document.getElementById('passwordGroup').style.display = 'none';
        document.getElementById('password').required = false;
        document.getElementById('userModal').style.display = 'block';
        
    } catch (error) {
        Toast.error(error.message, 'Failed to load user');
    }
}

async function deleteUser(id, username) {
    const confirmed = await Confirm.show({
        title: 'Delete User',
        message: `Are you sure you want to delete user "${username}"?`,
        confirmText: 'Delete',
        type: 'danger'
    });
    
    if (!confirmed) return;
    
    try {
        const response = await fetch(`/api/users/${id}`, {
            method: 'DELETE',
        });
        
        if (response.ok) {
            Toast.success('User deleted successfully!');
            loadUsers();
        } else {
            const error = await response.json();
            Toast.error(error.error || 'Unknown error', 'Failed to delete user');
        }
    } catch (error) {
        Toast.error(error.message, 'Failed to delete user');
    }
}

async function changePassword(id) {
    // Get current user to check if admin
    const currentUserResponse = await fetch('/api/users/me');
    const currentUser = await currentUserResponse.json();
    
    // Create modal dynamically
    const modal = document.createElement('div');
    modal.className = 'modal';
    modal.style.display = 'block';
    modal.id = 'changePasswordModal';
    
    const isOwnPassword = currentUser.id === id;
    
    modal.innerHTML = `
        <div class="modal-content">
            <span class="close" onclick="this.closest('.modal').remove()">&times;</span>
            <h3>${isOwnPassword ? 'Change Your Password' : 'Set New Password for User'}</h3>
            <form id="changePasswordFormDynamic">
                <input type="hidden" id="targetUserId" value="${id}">
                ${isOwnPassword ? `
                <div class="form-group">
                    <label for="currentPasswordInput">Current Password: *</label>
                    <input type="password" id="currentPasswordInput" name="current_password" class="form-input" required minlength="4">
                </div>
                ` : ''}
                <div class="form-group">
                    <label for="newPasswordInput">New Password: *</label>
                    <input type="password" id="newPasswordInput" name="new_password" class="form-input" required minlength="4" placeholder="Minimum 4 characters">
                </div>
                <div class="form-group">
                    <label for="confirmPasswordInput">Confirm Password: *</label>
                    <input type="password" id="confirmPasswordInput" name="confirm_password" class="form-input" required minlength="4" placeholder="Repeat new password">
                </div>
                <button type="submit" class="btn">Change Password</button>
            </form>
        </div>
    `;
    
    document.body.appendChild(modal);
    
    document.getElementById('changePasswordFormDynamic').addEventListener('submit', async (e) => {
        e.preventDefault();
        
        const userId = document.getElementById('targetUserId').value;
        const newPassword = document.getElementById('newPasswordInput').value;
        const confirmPassword = document.getElementById('confirmPasswordInput').value;
        const currentPasswordInput = document.getElementById('currentPasswordInput');
        
        if (newPassword !== confirmPassword) {
            showToast('Passwords do not match!', 'error');
            return;
        }
        
        if (newPassword.length < 4) {
            showToast('Password must be at least 4 characters long!', 'error');
            return;
        }
        
        const data = {
            new_password: newPassword,
        };
        
        // Add current password if changing own password
        if (currentPasswordInput) {
            data.current_password = currentPasswordInput.value;
        }
        
        try {
            const response = await apiFetch(`/api/users/${userId}/password`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(data)
            });
            
            const result = await response.json();
            showToast(result.message || 'Password changed successfully!', 'success');
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

document.getElementById('userForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const userId = document.getElementById('userId').value;
    const isEdit = userId !== '';
    
    const data = {
        username: document.getElementById('username').value,
        email: document.getElementById('email').value,
        role_id: parseInt(document.getElementById('roleId').value),
        enabled: document.getElementById('enabled').checked,
    };
    
    if (!isEdit) {
        data.password = document.getElementById('password').value;
    }
    
    try {
        const url = isEdit ? `/api/users/${userId}` : '/api/users';
        const method = isEdit ? 'PUT' : 'POST';
        
        const response = await fetch(url, {
            method: method,
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(data),
        });
        
        if (response.ok) {
            Toast.success(isEdit ? 'User updated successfully!' : 'User created successfully!');
            closeModal();
            loadUsers();
        } else {
            const error = await response.json();
            Toast.error(error.error || 'Unknown error', 'Failed to save user');
        }
    } catch (error) {
        Toast.error(error.message, 'Failed to save user');
    }
});

// Old password form handler removed - now using dynamic modal in changePassword() function

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

async function showRolePermissions(roleId, roleName, fromModal = null) {
    currentRoleId = roleId;
    currentRoleName = roleName;
    previousModal = fromModal; // Remember where we came from
    
    // Close any other open modals first
    closeModal();
    
    const modal = document.getElementById('rolePermissionsModal');
    const title = document.getElementById('rolePermissionsTitle');
    const content = document.getElementById('rolePermissionsContent');
    const actions = document.getElementById('rolePermissionsActions');
    
    title.textContent = `${roleName} - Permissions`;
    content.innerHTML = '<div class="loading">Loading permissions...</div>';
    actions.style.display = 'none';
    modal.style.display = 'block';
    
    try {
        const response = await fetch(`/api/roles/${roleId}`);
        if (!response.ok) {
            throw new Error('Failed to load role permissions');
        }
        
        const role = await response.json();
        const permissions = role.permissions || [];
        
        if (permissions.length === 0) {
            content.innerHTML = '<p style="padding: 1rem;">No permissions assigned to this role.</p>';
        } else {
            content.innerHTML = renderPermissionsList(permissions);
        }
        
        // Show edit button only if user can edit this role
        if (canEditRole(role)) {
            actions.style.display = 'block';
        }
        
    } catch (error) {
        content.innerHTML = `<div class="error" style="padding: 1rem;">Failed to load permissions: ${error.message}</div>`;
    }
}

function renderPermissionsList(permissions) {
    // Group permissions by category
    const grouped = {};
    permissions.forEach(perm => {
        if (!grouped[perm.category]) {
            grouped[perm.category] = [];
        }
        grouped[perm.category].push(perm);
    });
    
    let html = '<div style="padding: 1rem;">';
    
    Object.keys(grouped).sort().forEach(category => {
        html += `
            <div style="margin-bottom: 1.5rem;">
                <h4 style="color: #2c3e50; margin-bottom: 0.5rem; text-transform: capitalize; border-bottom: 2px solid #3498db; padding-bottom: 0.25rem;">
                    ${category}
                </h4>
                <ul style="list-style: none; padding-left: 0; margin-top: 0.5rem;">
        `;
        
        grouped[category].forEach(perm => {
            html += `
                <li style="padding: 0.5rem 0; border-bottom: 1px solid #ecf0f1;">
                    <strong style="color: #2c3e50;">${perm.display_name}</strong>
                    <br>
                    <small style="color: #7f8c8d;">${perm.description || perm.name}</small>
                </li>
            `;
        });
        
        html += `
                </ul>
            </div>
        `;
    });
    
    html += '</div>';
    return html;
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

async function editRolePermissions() {
    // First check if we have permission to edit this role
    try {
        const roleResponse = await fetch(`/api/roles/${currentRoleId}`);
        if (!roleResponse.ok) throw new Error('Failed to load role');
        const role = await roleResponse.json();
        
        if (!canEditRole(role)) {
            Toast.warning('You do not have permission to edit permissions for this role.');
            return;
        }
    } catch (error) {
        Toast.error(error.message, 'Failed to verify permissions');
        return;
    }
    
    // Close view modal
    document.getElementById('rolePermissionsModal').style.display = 'none';
    
    // Open edit modal
    const modal = document.getElementById('editRolePermissionsModal');
    const title = document.getElementById('editRolePermissionsTitle');
    const content = document.getElementById('editPermissionsContent');
    
    title.textContent = `Edit Permissions - ${currentRoleName}`;
    content.innerHTML = '<div class="loading">Loading...</div>';
    modal.style.display = 'block';
    
    try {
        // Load all permissions and current role permissions in parallel
        const [permResponse, roleResponse] = await Promise.all([
            fetch('/api/permissions'),
            fetch(`/api/roles/${currentRoleId}`)
        ]);
        
        if (!permResponse.ok || !roleResponse.ok) {
            throw new Error('Failed to load data');
        }
        
        allPermissions = await permResponse.json();
        const role = await roleResponse.json();
        const currentPermIds = new Set((role.permissions || []).map(p => p.id));
        
        // Group permissions by category
        const grouped = {};
        allPermissions.forEach(perm => {
            if (!grouped[perm.category]) {
                grouped[perm.category] = [];
            }
            grouped[perm.category].push(perm);
        });
        
        let html = '<div style="padding: 1rem;">';
        html += '<div style="margin-bottom: 1rem;"><button type="button" class="btn btn-sm" onclick="selectAllPermissions()">Select All</button> <button type="button" class="btn btn-sm" onclick="deselectAllPermissions()">Deselect All</button></div>';
        
        Object.keys(grouped).sort().forEach(category => {
            html += `
                <div style="margin-bottom: 1.5rem; background: #f8f9fa; padding: 1rem; border-radius: 4px;">
                    <h4 style="color: #2c3e50; margin-bottom: 0.75rem; text-transform: capitalize;">
                        <label style="cursor: pointer;">
                            <input type="checkbox" onchange="toggleCategory('${category}')" id="category_${category}"> 
                            ${category}
                        </label>
                    </h4>
                    <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(250px, 1fr)); gap: 0.5rem; margin-left: 1.5rem;">
            `;
            
            grouped[category].forEach(perm => {
                const checked = currentPermIds.has(perm.id) ? 'checked' : '';
                html += `
                    <label style="display: block; cursor: pointer; padding: 0.5rem; background: white; border-radius: 4px; border: 1px solid #dee2e6;">
                        <input type="checkbox" name="permissions" value="${perm.id}" ${checked} class="perm-checkbox category-${category}">
                        <strong>${perm.display_name}</strong>
                        <br>
                        <small style="color: #6c757d;">${perm.description || ''}</small>
                    </label>
                `;
            });
            
            html += `
                    </div>
                </div>
            `;
        });
        
        html += '</div>';
        content.innerHTML = html;
        
        // Update category checkboxes
        Object.keys(grouped).forEach(category => {
            updateCategoryCheckbox(category);
        });
        
    } catch (error) {
        content.innerHTML = `<div class="error" style="padding: 1rem;">Failed to load permissions: ${error.message}</div>`;
    }
}

function toggleCategory(category) {
    const categoryCheckbox = document.getElementById(`category_${category}`);
    const checkboxes = document.querySelectorAll(`.category-${category}`);
    checkboxes.forEach(cb => cb.checked = categoryCheckbox.checked);
}

function updateCategoryCheckbox(category) {
    const checkboxes = document.querySelectorAll(`.category-${category}`);
    const categoryCheckbox = document.getElementById(`category_${category}`);
    if (!categoryCheckbox) return;
    
    const total = checkboxes.length;
    const checked = Array.from(checkboxes).filter(cb => cb.checked).length;
    
    categoryCheckbox.checked = checked === total;
    categoryCheckbox.indeterminate = checked > 0 && checked < total;
}

function selectAllPermissions() {
    document.querySelectorAll('input[name="permissions"]').forEach(cb => cb.checked = true);
    // Update category checkboxes
    const categories = new Set();
    allPermissions.forEach(p => categories.add(p.category));
    categories.forEach(cat => updateCategoryCheckbox(cat));
}

function deselectAllPermissions() {
    document.querySelectorAll('input[name="permissions"]').forEach(cb => cb.checked = false);
    // Update category checkboxes
    const categories = new Set();
    allPermissions.forEach(p => categories.add(p.category));
    categories.forEach(cat => updateCategoryCheckbox(cat));
}

document.getElementById('editRolePermissionsForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const checkboxes = document.querySelectorAll('input[name="permissions"]:checked');
    const permissionIds = Array.from(checkboxes).map(cb => parseInt(cb.value));
    
    try {
        const response = await fetch(`/api/roles/${currentRoleId}/permissions`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ permission_ids: permissionIds }),
        });
        
        if (response.ok) {
            Toast.success('Permissions updated successfully!');
            closeModal();
            // Go back to permissions view or roles management
            if (previousModal === 'rolesManagement') {
                showRolePermissions(currentRoleId, currentRoleName, 'rolesManagement');
            } else {
                showRolePermissions(currentRoleId, currentRoleName);
            }
        } else {
            const error = await response.json();
            Toast.error(error.error || 'Unknown error', 'Failed to update permissions');
        }
    } catch (error) {
        Toast.error(error.message, 'Failed to update permissions');
    }
});

// Add event delegation for permission checkboxes to update category checkboxes
document.addEventListener('change', (e) => {
    if (e.target.classList.contains('perm-checkbox')) {
        const category = Array.from(e.target.classList).find(c => c.startsWith('category-'))?.replace('category-', '');
        if (category) {
            updateCategoryCheckbox(category);
        }
    }
});

// Role Management Functions
async function showRolesManagement() {
    // Check permission first
    if (!canViewRoles()) {
        Toast.warning('You do not have permission to view roles.');
        return;
    }
    
    const modal = document.getElementById('rolesManagementModal');
    const content = document.getElementById('rolesManagementContent');
    
    content.innerHTML = '<div class="loading">Loading roles...</div>';
    modal.style.display = 'block';
    
    try {
        const response = await fetch('/api/roles');
        if (!response.ok) throw new Error('Failed to load roles');
        
        const roles = await response.json();
        
        let html = '<table class="table"><thead><tr><th>Role Name</th><th>Display Name</th><th>Description</th><th>Type</th><th>Actions</th></tr></thead><tbody>';
        
        roles.forEach(role => {
            const typeBadge = role.is_super_admin ? '<span class="badge badge-danger">Super Admin</span>' : 
                             (role.is_system ? '<span class="badge badge-warning">System</span>' : 
                             '<span class="badge badge-info">Custom</span>');
            
            const canEdit = canEditRole(role);
            const canDelete = canDeleteRole(role);
            
            html += `
                <tr>
                    <td><strong>${escapeHtml(role.name)}</strong></td>
                    <td>${escapeHtml(role.display_name)}</td>
                    <td>${escapeHtml(role.description || '')}</td>
                    <td>${typeBadge}</td>
                    <td>
                        <button class="btn btn-sm" onclick="showRolePermissions(${role.id}, '${escapeHtml(role.display_name)}', 'rolesManagement')" title="View Permissions">Permissions</button>
                        ${canEdit ? `<button class="btn btn-sm" onclick="editRole(${role.id})">Edit</button>` : ''}
                        ${canDelete ? `<button class="btn btn-sm btn-danger" onclick="deleteRole(${role.id}, '${escapeHtml(role.name)}')">Delete</button>` : ''}
                    </td>
                </tr>
            `;
        });
        
        html += '</tbody></table>';
        
        // Add "Add New Role" button if user has permission
        if (canCreateRole()) {
            html += '<div style="margin-top: 1rem; text-align: right;"><button class="btn" onclick="showAddRoleModal()">Add New Role</button></div>';
        }
        
        content.innerHTML = html;
        
    } catch (error) {
        content.innerHTML = `<div class="error" style="padding: 1rem;">Failed to load roles: ${error.message}</div>`;
    }
}

function showAddRoleModal() {
    if (!canCreateRole()) {
        Toast.warning('You do not have permission to create roles.');
        return;
    }
    
    document.getElementById('roleModalTitle').textContent = 'Add New Role';
    document.getElementById('roleForm').reset();
    document.getElementById('roleFormId').value = '';
    document.getElementById('rolesManagementModal').style.display = 'none';
    document.getElementById('roleModal').style.display = 'block';
}

async function editRole(roleId) {
    try {
        const response = await fetch(`/api/roles/${roleId}`);
        if (!response.ok) throw new Error('Failed to load role');
        
        const role = await response.json();
        
        // Check permission before editing
        if (!canEditRole(role)) {
            Toast.warning('You do not have permission to edit this role.');
            return;
        }
        
        document.getElementById('roleModalTitle').textContent = 'Edit Role';
        document.getElementById('roleFormId').value = role.id;
        document.getElementById('roleName').value = role.name;
        document.getElementById('roleDisplayName').value = role.display_name;
        document.getElementById('roleDescription').value = role.description || '';
        
        document.getElementById('rolesManagementModal').style.display = 'none';
        document.getElementById('roleModal').style.display = 'block';
        
    } catch (error) {
        Toast.error(error.message, 'Failed to load role');
    }
}

async function deleteRole(roleId, roleName) {
    // First fetch the role to check permissions
    try {
        const roleResponse = await fetch(`/api/roles/${roleId}`);
        if (!roleResponse.ok) throw new Error('Failed to load role');
        const role = await roleResponse.json();
        
        if (!canDeleteRole(role)) {
            Toast.warning('You do not have permission to delete this role.');
            return;
        }
    } catch (error) {
        Toast.error(error.message, 'Failed to verify permissions');
        return;
    }
    
    const confirmed = await Confirm.show({
        title: 'Delete Role',
        message: `Are you sure you want to delete role "${roleName}"?\n\nAll users with this role will need to be reassigned.`,
        confirmText: 'Delete',
        type: 'danger'
    });
    
    if (!confirmed) return;
    
    try {
        const response = await fetch(`/api/roles/${roleId}`, {
            method: 'DELETE'
        });
        
        if (response.ok) {
            Toast.success('Role deleted successfully!');
            showRolesManagement();
        } else {
            const error = await response.json();
            Toast.error(error.error || 'Unknown error', 'Failed to delete role');
        }
    } catch (error) {
        Toast.error(error.message, 'Failed to delete role');
    }
}

document.getElementById('roleForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const roleId = document.getElementById('roleFormId').value;
    const data = {
        name: document.getElementById('roleName').value,
        display_name: document.getElementById('roleDisplayName').value,
        description: document.getElementById('roleDescription').value,
    };
    
    try {
        const url = roleId ? `/api/roles/${roleId}` : '/api/roles';
        const method = roleId ? 'PUT' : 'POST';
        
        const response = await fetch(url, {
            method: method,
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(data),
        });
        
        if (response.ok) {
            Toast.success(roleId ? 'Role updated successfully!' : 'Role created successfully!');
            closeModal();
            showRolesManagement();
            loadRoles(); // Refresh roles dropdown
        } else {
            const error = await response.json();
            Toast.error(error.error || 'Unknown error', 'Failed to save role');
        }
    } catch (error) {
        Toast.error(error.message, 'Failed to save role');
    }
});

// Initialize
async function init() {
    await loadCurrentUser();
    await loadRoles();
    await loadUsers();
    
    // Show/hide Manage Roles button based on permissions
    const manageRolesBtn = document.querySelector('button[onclick="showRolesManagement()"]');
    if (manageRolesBtn) {
        if (!canViewRoles()) {
            manageRolesBtn.style.display = 'none';
        }
    }
    
    // Show/hide Add User button based on permissions
    const addUserBtn = document.querySelector('button[onclick="showAddUserModal()"]');
    if (addUserBtn) {
        if (!hasPermission('users.create') && !currentUser.role.is_super_admin) {
            addUserBtn.style.display = 'none';
        }
    }
}

init();

function goBackToRolesManagement() {
    closeModal();
    if (previousModal === 'rolesManagement') {
        showRolesManagement();
    }
}
