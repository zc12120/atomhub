import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import LoginPage from './LoginPage';

vi.mock('../auth', () => ({
  isAuthenticated: () => false,
  loginAndFetchSession: vi.fn()
}));

describe('LoginPage', () => {
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
});
