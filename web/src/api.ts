export interface DashboardItem {
  model: string;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  request_count: number;
}

export interface DashboardSummary {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  request_count?: number;
}

export interface DashboardResponse {
  items: DashboardItem[];
  summary: DashboardSummary;
  health?: {
    healthy_keys: number;
    unhealthy_keys: number;
    total_keys: number;
  };
}

export interface AdminSession {
  authenticated: boolean;
  username?: string;
}

export interface AdminKey {
  id: number;
  provider: string;
  label: string;
  status: string;
  base_url?: string;
  enabled: boolean;
  last_error?: string;
  last_used_at?: string;
}

export interface CreateKeyPayload {
  name: string;
  provider: string;
  base_url: string;
  api_key: string;
  enabled?: boolean;
}

export interface UpdateKeyPayload {
  name?: string;
  provider?: string;
  base_url?: string;
  api_key?: string;
  enabled?: boolean;
}

export interface KeysResponse {
  items: AdminKey[];
}

export interface AdminModel {
  model: string;
  provider: string;
  key_count: number;
  healthy_keys: number;
}

export interface ModelsResponse {
  items: AdminModel[];
}

export interface HealthSummary {
  healthy_keys: number;
  unhealthy_keys: number;
  total_keys: number;
}

export interface HealthResponse {
  summary: HealthSummary;
  keys: AdminKey[];
}

export interface AdminRequestLog {
  id: number;
  key_id: number;
  key_label?: string;
  provider?: string;
  model: string;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  latency_ms: number;
  status: string;
  error_message?: string;
  created_at: string;
}

export interface RequestsSummary {
  request_count: number;
  error_count: number;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
}

export interface RequestsResponse {
  items: AdminRequestLog[];
  summary: RequestsSummary;
  filters: {
    model?: string;
    models: string[];
  };
}

interface LoginPayload {
  username: string;
  password: string;
}

interface RequestOptions extends RequestInit {
  skipJson?: boolean;
}

const defaultHeaders = {
  'Content-Type': 'application/json'
};

async function requestJson<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const response = await fetch(path, {
    credentials: 'include',
    ...options,
    headers: {
      ...defaultHeaders,
      ...(options.headers ?? {})
    }
  });

  if (!response.ok) {
    let message = `Request failed: ${response.status}`;
    try {
      const errorBody = (await response.json()) as { error?: string };
      if (errorBody.error) {
        message = errorBody.error;
      }
    } catch {
      // Ignore parsing errors and keep fallback message.
    }
    throw new Error(message);
  }

  if (options.skipJson || response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

export const api = {
  login(payload: LoginPayload): Promise<{ ok?: boolean; username?: string }> {
    return requestJson('/admin/login', {
      method: 'POST',
      body: JSON.stringify(payload)
    });
  },

  logout(): Promise<void> {
    return requestJson('/admin/logout', {
      method: 'POST',
      skipJson: true
    });
  },

  getSession(): Promise<AdminSession> {
    return requestJson('/admin/session');
  },

  getDashboard(): Promise<DashboardResponse> {
    return requestJson('/admin/dashboard');
  },

  getKeys(): Promise<KeysResponse> {
    return requestJson('/admin/keys');
  },

  createKey(payload: CreateKeyPayload): Promise<AdminKey> {
    return requestJson('/admin/keys', {
      method: 'POST',
      body: JSON.stringify(payload)
    });
  },

  updateKey(id: number, payload: UpdateKeyPayload): Promise<AdminKey> {
    return requestJson(`/admin/keys/${id}`, {
      method: 'PUT',
      body: JSON.stringify(payload)
    });
  },

  deleteKey(id: number): Promise<void> {
    return requestJson(`/admin/keys/${id}`, {
      method: 'DELETE',
      skipJson: true
    });
  },

  probeKey(id: number): Promise<AdminKey> {
    return requestJson(`/admin/keys/${id}/probe`, {
      method: 'POST'
    });
  },

  getModels(): Promise<ModelsResponse> {
    return requestJson('/admin/models');
  },

  getHealth(): Promise<HealthResponse> {
    return requestJson('/admin/health');
  },

  getRequests(model?: string): Promise<RequestsResponse> {
    const query = model ? `?model=${encodeURIComponent(model)}` : '';
    return requestJson(`/admin/requests${query}`);
  }
};
