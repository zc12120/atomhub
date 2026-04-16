import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import LoginPage from './LoginPage';
import { loginAndFetchSession } from '../auth';

vi.mock('../auth', () => ({
  isAuthenticated: () => false,
  loginAndFetchSession: vi.fn()
}));

describe('LoginPage', () => {
  afterEach(() => {
    cleanup();
  });

  it('renders chinese login copy', () => {
    render(
      <MemoryRouter>
        <LoginPage session={{ authenticated: false }} onLogin={() => {}} />
      </MemoryRouter>
    );

    expect(screen.getByRole('heading', { name: '管理员登录' })).toBeInTheDocument();
    expect(screen.getByText('登录后可查看仪表盘、密钥状态与请求记录。')).toBeInTheDocument();
    expect(screen.getByLabelText('用户名')).toBeInTheDocument();
    expect(screen.getByLabelText('密码')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '登录' })).toBeInTheDocument();
  });

  it('shows chinese error copy when login fails', async () => {
    vi.mocked(loginAndFetchSession).mockRejectedValueOnce(new Error('invalid credentials'));

    render(
      <MemoryRouter>
        <LoginPage session={{ authenticated: false }} onLogin={() => {}} />
      </MemoryRouter>
    );

    fireEvent.change(screen.getByLabelText('密码'), { target: { value: 'wrong' } });
    fireEvent.click(screen.getByRole('button', { name: '登录' }));

    await waitFor(() => {
      expect(screen.getByText('用户名或密码错误。')).toBeInTheDocument();
    });
  });
});
