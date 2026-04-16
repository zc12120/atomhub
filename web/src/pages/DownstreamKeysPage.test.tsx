import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import DownstreamKeysPage from './DownstreamKeysPage';
import { api } from '../api';

vi.mock('../api', () => ({
  api: {
    getDownstreamKeys: vi.fn(),
    createDownstreamKey: vi.fn(),
    updateDownstreamKey: vi.fn(),
    deleteDownstreamKey: vi.fn()
  }
}));

const baseItem = {
  id: 1,
  name: '渠道 A',
  token_prefix: 'atom_abcd1234',
  enabled: true,
  last_used_at: '2026-04-16T00:00:00Z',
  request_count: 8,
  prompt_tokens: 120,
  completion_tokens: 30,
  total_tokens: 150,
  created_at: '2026-04-15T00:00:00Z',
  updated_at: '2026-04-16T00:00:00Z'
};

describe('DownstreamKeysPage', () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.getDownstreamKeys).mockResolvedValue({ items: [baseItem] });
    vi.mocked(api.createDownstreamKey).mockResolvedValue({
      item: {
        ...baseItem,
        id: 2,
        name: '渠道 B',
        enabled: false
      },
      token: 'atom_token_plaintext_once'
    });
    vi.mocked(api.updateDownstreamKey).mockResolvedValue({
      ...baseItem,
      enabled: false
    });
    vi.mocked(api.deleteDownstreamKey).mockResolvedValue();
  });

  it('renders list and key statistics in chinese', async () => {
    render(<DownstreamKeysPage />);

    expect(await screen.findByRole('heading', { name: '下游密钥' })).toBeInTheDocument();
    expect(screen.getByText('atom_abcd1234')).toBeInTheDocument();
    expect(screen.getByText('150')).toBeInTheDocument();
    expect(screen.getByText('8')).toBeInTheDocument();
    expect(screen.getByText('启用中')).toBeInTheDocument();
  });

  it('creates downstream key and shows plaintext token once', async () => {
    render(<DownstreamKeysPage />);

    await screen.findByText('渠道 A');

    fireEvent.change(screen.getByLabelText('名称'), { target: { value: '渠道 B' } });
    fireEvent.change(screen.getByLabelText('创建后立即启用'), { target: { value: 'false' } });

    fireEvent.click(screen.getByRole('button', { name: '创建下游密钥' }));

    await waitFor(() => {
      expect(api.createDownstreamKey).toHaveBeenCalledWith({ name: '渠道 B', enabled: false });
    });

    expect(screen.getByText(/请立即复制并妥善保存此 token/i)).toBeInTheDocument();
    expect(screen.getByDisplayValue('atom_token_plaintext_once')).toBeInTheDocument();
    expect(api.getDownstreamKeys).toHaveBeenCalledTimes(2);
  });

  it('toggles enable state and deletes a key from row actions', async () => {
    render(<DownstreamKeysPage />);

    await screen.findByText('渠道 A');

    fireEvent.click(screen.getByRole('button', { name: '停用' }));

    await waitFor(() => {
      expect(api.updateDownstreamKey).toHaveBeenCalledWith(1, { enabled: false });
    });

    fireEvent.click(screen.getByRole('button', { name: '删除' }));

    await waitFor(() => {
      expect(api.deleteDownstreamKey).toHaveBeenCalledWith(1);
    });
    expect(api.getDownstreamKeys).toHaveBeenCalledTimes(3);
  });
});
