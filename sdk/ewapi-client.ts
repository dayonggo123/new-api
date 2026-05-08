/**
 * ewapi - new-api 前端 SDK
 * 封装用户积分、签到、消息通知、提示词解锁等接口
 */

import axios, { AxiosInstance } from 'axios';

// ==================== 类型定义 ====================

export interface UserPoints {
  total_points: number;
  consecutive_days: number;
  last_signin_date: number;
  today_signed: boolean;
  next_signin_points: number;
}

export interface SigninResult {
  points_earned: number;
  bonus_points: number;
  total_points: number;
  consecutive_days: number;
}

export interface SigninHistoryItem {
  signin_date: number;
  points: number;
  bonus_points: number;
  total_points_after: number;
}

export interface NotificationItem {
  id: number;
  user_id: number;
  title: string;
  content: string;
  type: 'system' | 'promotion' | 'announcement' | 'task_status';
  is_read: boolean;
  action_url: string;
  created_time: number;
}

export interface UnlockPromptResult {
  prompt_id: number;
  cost: number;
  remaining_points: number;
}

export interface UnlockedPromptItem {
  prompt_id: number;
  title: string;
  cover_image_url: string;
  cost: number;
  unlocked_at: number;
}

export interface PromptItem {
  id: number;
  category_id: number;
  title: string;
  content: string;
  cover_image_url: string;
  author: string;
  tags: string;
  is_premium: boolean;
  unlock_cost: number;
  category_name?: string;
}

export interface PageResult<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface ApiResponse<T> {
  success: boolean;
  message: string;
  data: T;
}

// ==================== SDK 客户端 ====================

export class EwapiClient {
  private client: AxiosInstance;
  private baseURL: string;

  constructor(baseURL: string, token?: string) {
    this.baseURL = baseURL.replace(/\/$/, '');
    this.client = axios.create({
      baseURL: this.baseURL,
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (token) {
      this.setToken(token);
    }

    // 响应拦截器：统一错误处理
    this.client.interceptors.response.use(
      (response) => response.data,
      (error) => {
        if (error.response?.data?.message) {
          return Promise.reject(new Error(error.response.data.message));
        }
        return Promise.reject(error);
      }
    );
  }

  /** 设置/更新 Token */
  setToken(token: string) {
    this.client.defaults.headers.common['Authorization'] = `Bearer ${token}`;
  }

  /** 设置用户 ID（用于 New-Api-User Header） */
  setUserId(userId: number) {
    this.client.defaults.headers.common['New-Api-User'] = String(userId);
  }

  /** 清除 Token */
  clearToken() {
    delete this.client.defaults.headers.common['Authorization'];
    delete this.client.defaults.headers.common['New-Api-User'];
  }

  // ==================== 用户积分 & 签到 ====================

  /** 获取用户积分和签到状态 */
  async getUserPoints(): Promise<UserPoints> {
    const res = await this.client.get<ApiResponse<UserPoints>>('/api/user/points');
    return res.data;
  }

  /** 执行签到 */
  async signin(): Promise<SigninResult> {
    const res = await this.client.post<ApiResponse<SigninResult>>('/api/user/signin');
    return res.data;
  }

  /** 获取签到历史 */
  async getSigninHistory(page = 1, pageSize = 20): Promise<PageResult<SigninHistoryItem>> {
    const res = await this.client.get<ApiResponse<PageResult<SigninHistoryItem>>>('/api/user/signin-history', {
      params: { page, page_size: pageSize },
    });
    return res.data;
  }

  // ==================== 提示词解锁 ====================

  /** 解锁提示词 */
  async unlockPrompt(promptId: number): Promise<UnlockPromptResult> {
    const res = await this.client.post<ApiResponse<UnlockPromptResult>>('/api/user/unlock-prompt', {
      prompt_id: promptId,
    });
    return res.data;
  }

  /** 获取已解锁提示词列表 */
  async getUnlockedPrompts(page = 1, pageSize = 20): Promise<PageResult<UnlockedPromptItem>> {
    const res = await this.client.get<ApiResponse<PageResult<UnlockedPromptItem>>>('/api/user/unlocked-prompts', {
      params: { page, page_size: pageSize },
    });
    return res.data;
  }

  /** 检查提示词是否已解锁 */
  async isPromptUnlocked(promptId: number, unlockedPrompts?: UnlockedPromptItem[]): Promise<boolean> {
    if (unlockedPrompts) {
      return unlockedPrompts.some((p) => p.prompt_id === promptId);
    }
    const list = await this.getUnlockedPrompts(1, 1000);
    return list.items.some((p) => p.prompt_id === promptId);
  }

  // ==================== 消息通知 ====================

  /** 获取消息列表 */
  async getNotifications(page = 1, pageSize = 20): Promise<PageResult<NotificationItem>> {
    const res = await this.client.get<ApiResponse<PageResult<NotificationItem>>>('/api/notifications', {
      params: { page, page_size: pageSize },
    });
    return res.data;
  }

  /** 获取未读消息数 */
  async getUnreadCount(): Promise<number> {
    const res = await this.client.get<ApiResponse<{ unread_count: number }>>('/api/notifications/unread-count');
    return res.data.unread_count;
  }

  /** 标记单条消息已读 */
  async markAsRead(notificationId: number): Promise<void> {
    await this.client.post(`/api/notifications/${notificationId}/read`);
  }

  /** 标记全部消息已读 */
  async markAllAsRead(): Promise<{ marked_count: number }> {
    const res = await this.client.post<ApiResponse<{ marked_count: number }>>('/api/notifications/read-all');
    return res.data;
  }

  // ==================== 公开提示词 ====================

  /** 获取公开提示词列表 */
  async getPublicPrompts(params?: {
    category_id?: number;
    keyword?: string;
    page?: number;
    page_size?: number;
  }): Promise<PageResult<PromptItem>> {
    const res = await this.client.get<ApiResponse<PageResult<PromptItem>>>('/api/public/prompts', { params });
    return res.data;
  }

  /** 获取公开提示词详情 */
  async getPublicPrompt(promptId: number): Promise<PromptItem> {
    const res = await this.client.get<ApiResponse<PromptItem>>(`/api/public/prompts/${promptId}`);
    return res.data;
  }

  // ==================== 登录 & Token ====================

  /**
   * 登录（用户名密码）
   * 登录成功后需手动调用 setToken()
   */
  async login(username: string, password: string): Promise<{ token: string; user: any }> {
    const res = await this.client.post<ApiResponse<any>>('/api/user/login', {
      username,
      password,
    });
    if (res.data?.token) {
      this.setToken(res.data.token);
    }
    return res.data;
  }

  /** 获取当前登录用户信息 */
  async getSelf(): Promise<any> {
    const res = await this.client.get<ApiResponse<any>>('/api/user/self');
    return res.data;
  }
}

// ==================== React Hook 示例（可选）====================

/*
import { useState, useEffect, useCallback } from 'react';

// 全局 SDK 实例
let clientInstance: EwapiClient | null = null;

export function initClient(baseURL: string, token?: string) {
  clientInstance = new EwapiClient(baseURL, token);
  return clientInstance;
}

export function getClient(): EwapiClient {
  if (!clientInstance) {
    throw new Error('EwapiClient not initialized. Call initClient() first.');
  }
  return clientInstance;
}

// Hook: 用户积分
export function useUserPoints() {
  const [points, setPoints] = useState<UserPoints | null>(null);
  const [loading, setLoading] = useState(false);

  const fetch = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getClient().getUserPoints();
      setPoints(data);
    } finally {
      setLoading(false);
    }
  }, []);

  const signin = useCallback(async () => {
    const result = await getClient().signin();
    setPoints((prev) =>
      prev
        ? {
            ...prev,
            total_points: result.total_points,
            consecutive_days: result.consecutive_days,
            today_signed: true,
          }
        : null
    );
    return result;
  }, []);

  useEffect(() => {
    fetch();
  }, [fetch]);

  return { points, loading, refresh: fetch, signin };
}

// Hook: 消息通知
export function useNotifications() {
  const [unreadCount, setUnreadCount] = useState(0);

  const fetchUnread = useCallback(async () => {
    const count = await getClient().getUnreadCount();
    setUnreadCount(count);
  }, []);

  const markAllRead = useCallback(async () => {
    await getClient().markAllAsRead();
    setUnreadCount(0);
  }, []);

  useEffect(() => {
    fetchUnread();
  }, [fetchUnread]);

  return { unreadCount, refresh: fetchUnread, markAllRead };
}
*/

export default EwapiClient;
