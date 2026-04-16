import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import HealthPage from './HealthPage';

const mockData = {
  summary: { healthy_keys: 2, unhealthy_keys: 1, total_keys: 3 },
  keys: [{ id: 1, label: '主密钥', provider: 'openai', status: 'healthy', last_error: '' }]
};

describe('HealthPage', () => {
  it('renders chinese health copy', () => {
    render(<HealthPage data={mockData as never} />);

    expect(screen.getByRole('heading', { name: '健康状态' })).toBeInTheDocument();
    expect(screen.getByText('健康密钥')).toBeInTheDocument();
    expect(screen.getByText('异常密钥')).toBeInTheDocument();
    expect(screen.getByText('密钥总数')).toBeInTheDocument();
  });
});
