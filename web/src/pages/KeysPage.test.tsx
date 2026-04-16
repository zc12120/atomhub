import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import KeysPage from './KeysPage';
import { api } from '../api';

vi.mock('../api', () => ({
  api: {
    getKeys: vi.fn(),
    createKey: vi.fn(),
    updateKey: vi.fn(),
    deleteKey: vi.fn(),
    probeKey: vi.fn()
  }
}));

const keys = [
  {
    id: 1,
    provider: 'openai',
    label: 'Primary key',
    status: 'healthy',
    base_url: 'https://api.openai.com',
    enabled: true,
    last_error: '',
    last_used_at: '2026-04-16T00:00:00Z'
  }
];

describe('KeysPage', () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.getKeys).mockResolvedValue({ items: keys });
    vi.mocked(api.createKey).mockResolvedValue(keys[0]);
    vi.mocked(api.updateKey).mockResolvedValue(keys[0]);
    vi.mocked(api.deleteKey).mockResolvedValue();
    vi.mocked(api.probeKey).mockResolvedValue(keys[0]);
  });

  it('toggles key enabled state from the row actions', async () => {
    render(<KeysPage />);

    expect(await screen.findByText('Primary key')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /停用/i }));

    await waitFor(() => {
      expect(api.updateKey).toHaveBeenCalledWith(1, { enabled: false });
    });
    expect(api.getKeys).toHaveBeenCalledTimes(2);
  });

  it('allows editing key metadata from an inline editor', async () => {
    render(<KeysPage />);

    expect(await screen.findByText('Primary key')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /编辑/i }));

    const labelInput = screen.getByLabelText('编辑名称');
    const baseURLInput = screen.getByLabelText('编辑 Base URL');
    const apiKeyInput = screen.getByLabelText('新的 API Key');

    fireEvent.change(labelInput, { target: { value: 'Renamed key' } });
    fireEvent.change(baseURLInput, { target: { value: 'https://proxy.example.com' } });
    fireEvent.change(apiKeyInput, { target: { value: 'sk-updated' } });

    fireEvent.click(screen.getByRole('button', { name: /保存修改/i }));

    await waitFor(() => {
      expect(api.updateKey).toHaveBeenCalledWith(1, {
        name: 'Renamed key',
        provider: 'openai',
        base_url: 'https://proxy.example.com',
        api_key: 'sk-updated'
      });
    });
  });
});
