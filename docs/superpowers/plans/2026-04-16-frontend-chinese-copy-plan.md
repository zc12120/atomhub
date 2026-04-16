# AtomHub 前端中文化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 AtomHub 前端所有用户可见文案改成中文，并保留 `token` 一词不翻译。

**Architecture:** 本次不引入 i18n 或文案常量层，直接在现有 React 页面和布局组件里替换可见文本，同时同步更新受影响的前端测试断言。URL、接口字段、后端逻辑保持不变，只改前端展示层。

**Tech Stack:** React 18、TypeScript、React Router、Vitest、Testing Library、Vite

---

### Task 1: 全局框架与登录页中文化

**Files:**
- Modify: `web/src/App.tsx`
- Modify: `web/src/components/Layout.tsx`
- Modify: `web/src/pages/LoginPage.tsx`
- Create: `web/src/pages/LoginPage.test.tsx`

- [ ] **Step 1: 先写登录页中文文案测试（红灯）**

```tsx
import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import LoginPage from './LoginPage';

vi.mock('../auth', () => ({
  isAuthenticated: () => false,
  loginAndFetchSession: vi.fn()
}));

describe('LoginPage', () => {
  it('渲染中文登录文案', () => {
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
```

- [ ] **Step 2: 运行登录页测试，确认失败**

Run:
```bash
cd /mnt/f/4-15/.worktrees/frontend-zh/web
npm test -- --run LoginPage
```

Expected: FAIL，提示找不到中文标题或中文按钮文案。

- [ ] **Step 3: 实现登录页中文文案**

将 `web/src/pages/LoginPage.tsx` 中的可见文本改成下面这种形式：

```tsx
<h1>管理员登录</h1>
<p className="muted">登录后可查看仪表盘、密钥状态与请求记录。</p>

<label htmlFor="username">用户名</label>
<label htmlFor="password">密码</label>

<button type="submit" disabled={submitting}>
  {submitting ? '登录中…' : '登录'}
</button>
```

错误提示改成中文：

```tsx
setError('登录成功，但未检测到有效会话。');
const message = submitError instanceof Error ? submitError.message : '登录失败。';
```

- [ ] **Step 4: 将全局框架文案改成中文**

在 `web/src/App.tsx` 和 `web/src/components/Layout.tsx` 中改成：

```tsx
<p className="muted">正在检查管理员会话…</p>
```

```tsx
const navItems = [
  { to: '/dashboard', label: '仪表盘' },
  { to: '/keys', label: '密钥' },
  { to: '/models', label: '模型' },
  { to: '/requests', label: '请求记录' },
  { to: '/health', label: '健康状态' }
];
```

```tsx
<h1>AtomHub 管理后台</h1>
<p className="topbar-subtitle">已登录{username ? `：${username}` : ''}</p>
<button type="button" className="secondary-button" onClick={handleLogout}>退出登录</button>
<aside className="sidebar" aria-label="管理后台导航">
```

- [ ] **Step 5: 重新运行登录页测试**

Run:
```bash
cd /mnt/f/4-15/.worktrees/frontend-zh/web
npm test -- --run LoginPage
```

Expected: PASS

- [ ] **Step 6: 提交这一批改动**

```bash
git add web/src/App.tsx web/src/components/Layout.tsx web/src/pages/LoginPage.tsx web/src/pages/LoginPage.test.tsx
git commit -m "feat: localize shell and login page to chinese"
```

---

### Task 2: 仪表盘、模型页、健康状态页中文化

**Files:**
- Modify: `web/src/pages/DashboardPage.tsx`
- Modify: `web/src/pages/HealthPage.tsx`
- Modify: `web/src/pages/ModelsPage.tsx`
- Modify: `web/src/pages/DashboardPage.test.tsx`
- Create: `web/src/pages/HealthPage.test.tsx`
- Create: `web/src/pages/ModelsPage.test.tsx`

- [ ] **Step 1: 先补健康状态页与模型页中文测试（红灯）**

`web/src/pages/HealthPage.test.tsx`

```tsx
import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import HealthPage from './HealthPage';

const mockData = {
  summary: { healthy_keys: 2, unhealthy_keys: 1, total_keys: 3 },
  keys: [{ id: 1, label: '主密钥', provider: 'openai', status: 'healthy', last_error: '' }]
};

describe('HealthPage', () => {
  it('渲染中文健康状态文案', () => {
    render(<HealthPage data={mockData as never} />);
    expect(screen.getByRole('heading', { name: '健康状态' })).toBeInTheDocument();
    expect(screen.getByText('健康密钥')).toBeInTheDocument();
    expect(screen.getByText('异常密钥')).toBeInTheDocument();
    expect(screen.getByText('密钥总数')).toBeInTheDocument();
  });
});
```

`web/src/pages/ModelsPage.test.tsx`

```tsx
import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import ModelsPage from './ModelsPage';

const mockData = {
  items: [{ model: 'gpt-4o-mini', provider: 'openai', key_count: 2, healthy_keys: 2 }]
};

describe('ModelsPage', () => {
  it('渲染中文模型页文案', () => {
    render(<ModelsPage data={mockData as never} />);
    expect(screen.getByRole('heading', { name: '模型' })).toBeInTheDocument();
    expect(screen.getByText('可用密钥数')).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: 更新仪表盘测试为中文断言（红灯）**

把 `web/src/pages/DashboardPage.test.tsx` 中的断言改成：

```tsx
expect(screen.getByRole('heading', { name: '仪表盘' })).toBeInTheDocument();
expect(screen.getByText('Prompt token')).toBeInTheDocument();
expect(screen.getByText('Completion token')).toBeInTheDocument();
expect(screen.getByText('Total token')).toBeInTheDocument();
```

- [ ] **Step 3: 运行这三个页面测试，确认失败**

Run:
```bash
cd /mnt/f/4-15/.worktrees/frontend-zh/web
npm test -- --run DashboardPage HealthPage ModelsPage
```

Expected: FAIL，原因是页面仍为英文文案。

- [ ] **Step 4: 实现三个页面的中文文案**

在 `web/src/pages/DashboardPage.tsx` 中替换为：

```tsx
<h2>仪表盘</h2>
{loading ? <p className="muted">正在加载用量汇总…</p> : null}
const message = loadError instanceof Error ? loadError.message : '加载仪表盘失败。';
<StatCard label="Prompt token" value={dashboard.summary.prompt_tokens} />
<StatCard label="Completion token" value={dashboard.summary.completion_tokens} />
<StatCard label="Total token" value={dashboard.summary.total_tokens} />
<th>模型</th>
<th>请求数</th>
```

在 `web/src/pages/HealthPage.tsx` 中替换为：

```tsx
<h2>健康状态</h2>
{loading ? <p className="muted">正在加载健康状态…</p> : null}
<StatCard label="健康密钥" value={health.summary.healthy_keys} />
<StatCard label="异常密钥" value={health.summary.unhealthy_keys} />
<StatCard label="密钥总数" value={health.summary.total_keys} />
<th>名称</th>
<th>提供商</th>
<th>状态</th>
<th>最近错误</th>
```

在 `web/src/pages/ModelsPage.tsx` 中替换为：

```tsx
<h2>模型</h2>
{loading ? <p className="muted">正在加载模型列表…</p> : null}
const message = loadError instanceof Error ? loadError.message : '加载模型列表失败。';
<th>模型</th>
<th>提供商</th>
<th>密钥数</th>
<th>可用密钥数</th>
```

- [ ] **Step 5: 重新运行页面测试**

Run:
```bash
cd /mnt/f/4-15/.worktrees/frontend-zh/web
npm test -- --run DashboardPage HealthPage ModelsPage
```

Expected: PASS

- [ ] **Step 6: 提交这一批改动**

```bash
git add web/src/pages/DashboardPage.tsx web/src/pages/HealthPage.tsx web/src/pages/ModelsPage.tsx web/src/pages/DashboardPage.test.tsx web/src/pages/HealthPage.test.tsx web/src/pages/ModelsPage.test.tsx
git commit -m "feat: localize dashboard models and health pages to chinese"
```

---

### Task 3: 密钥页与请求记录页中文化

**Files:**
- Modify: `web/src/pages/KeysPage.tsx`
- Modify: `web/src/pages/RequestsPage.tsx`
- Modify: `web/src/pages/KeysPage.test.tsx`
- Modify: `web/src/pages/RequestsPage.test.tsx`

- [ ] **Step 1: 先把 Keys/Requests 测试断言改成中文（红灯）**

在 `web/src/pages/KeysPage.test.tsx` 中将按钮与标签断言改成中文，例如：

```tsx
expect(screen.getByRole('button', { name: /停用/i })).toBeInTheDocument();
expect(screen.getByRole('button', { name: /编辑/i })).toBeInTheDocument();
expect(screen.getByLabelText('编辑名称')).toBeInTheDocument();
expect(screen.getByLabelText('编辑 Base URL')).toBeInTheDocument();
expect(screen.getByLabelText('新的 API Key')).toBeInTheDocument();
expect(screen.getByRole('button', { name: /保存修改/i })).toBeInTheDocument();
```

在 `web/src/pages/RequestsPage.test.tsx` 中改成：

```tsx
expect(screen.getByRole('heading', { name: '请求记录' })).toBeInTheDocument();
expect(screen.getByRole('combobox', { name: /模型筛选/i })).toBeInTheDocument();
```

- [ ] **Step 2: 运行 Keys/Requests 测试，确认失败**

Run:
```bash
cd /mnt/f/4-15/.worktrees/frontend-zh/web
npm test -- --run KeysPage RequestsPage
```

Expected: FAIL，原因是页面仍使用英文按钮、标签和标题。

- [ ] **Step 3: 将密钥页文案改成中文**

在 `web/src/pages/KeysPage.tsx` 中替换：

```tsx
<h2>密钥</h2>
<h3>新增上游密钥</h3>
<label>名称</label>
<label>提供商</label>
<label>Base URL</label>
<label className="full-width">API Key</label>
<button type="submit" disabled={submitting}>{submitting ? '保存中…' : '保存密钥'}</button>
```

表格与按钮改为：

```tsx
<th>名称</th>
<th>提供商</th>
<th>状态</th>
<th>启用</th>
<th>最近使用</th>
<th>最近错误</th>
<th>操作</th>

<button type="button" className="secondary-button">编辑</button>
<button type="button" className="secondary-button">{item.enabled ? '停用' : '启用'}</button>
<button type="button" className="secondary-button">探测</button>
<button type="button" className="danger-button">删除</button>
```

行内编辑区改为：

```tsx
<label>编辑名称</label>
<label>编辑提供商</label>
<label>编辑 Base URL</label>
<label>新的 API Key</label>
<button type="button">保存修改</button>
<button type="button" className="secondary-button">取消</button>
```

- [ ] **Step 4: 将请求记录页文案改成中文，但保留 token**

在 `web/src/pages/RequestsPage.tsx` 中替换：

```tsx
<h2>请求记录</h2>
<label htmlFor="request-model-filter">模型筛选</label>
<option value="">全部模型</option>
{loading ? <p className="muted">正在加载最近请求…</p> : null}
const message = loadError instanceof Error ? loadError.message : '加载请求记录失败。';

<StatCard label="请求数" value={summary.request_count} />
<StatCard label="错误数" value={summary.error_count} />
<StatCard label="Prompt token" value={summary.prompt_tokens} />
<StatCard label="Completion token" value={summary.completion_tokens} />
<StatCard label="Total token" value={summary.total_tokens} />
```

表头改成：

```tsx
<th>模型</th>
<th>请求数</th>
<th>Total token</th>
<th>占比</th>

<th>时间</th>
<th>模型</th>
<th>密钥</th>
<th>提供商</th>
<th>状态</th>
<th>延迟</th>
<th>Total token</th>
<th>错误</th>
```

空状态改成：

```tsx
没有符合当前筛选条件的请求记录。
```

- [ ] **Step 5: 重新运行 Keys/Requests 测试**

Run:
```bash
cd /mnt/f/4-15/.worktrees/frontend-zh/web
npm test -- --run KeysPage RequestsPage
```

Expected: PASS

- [ ] **Step 6: 提交这一批改动**

```bash
git add web/src/pages/KeysPage.tsx web/src/pages/RequestsPage.tsx web/src/pages/KeysPage.test.tsx web/src/pages/RequestsPage.test.tsx
git commit -m "feat: localize keys and requests pages to chinese"
```

---

### Task 4: 全量前端验收

**Files:**
- Verify only: `web/src/**/*`

- [ ] **Step 1: 全局扫一遍是否还残留英文可见文案**

重点检查这些文件：

```bash
cd /mnt/f/4-15/.worktrees/frontend-zh
rg -n 'Checking admin session|Admin Login|Dashboard|Keys|Models|Requests|Health|Loading|Save key|Probe|Delete|Edit|Log out|Sign in' web/src
```

Expected: 不再出现这些前端英文可见文案；允许保留 `token`、`API Key`、`Base URL`。

- [ ] **Step 2: 跑完整前端测试**

Run:
```bash
cd /mnt/f/4-15/.worktrees/frontend-zh/web
npm test
```

Expected: PASS（3+ test files，全部通过）

- [ ] **Step 3: 跑前端构建**

Run:
```bash
cd /mnt/f/4-15/.worktrees/frontend-zh/web
npm run build
```

Expected: PASS，Vite build 成功。

- [ ] **Step 4: 最终提交**

```bash
git add web/src
git commit -m "feat: localize frontend copy to chinese"
```

