/*
Copyright (C) 2025 QuantumNous

自动翻译开关组件 - 可放置在提示词/GEO/文章管理页面
*/

import React, { useState, useEffect, useCallback } from 'react';
import { Switch, Tag, Banner, Space, Typography } from '@douyinfe/semi-ui';
import { IconPlay, IconPause, IconAlertTriangle } from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '@/helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

/**
 * AutoTranslateToggle - 系统自动翻译开关
 * 放在管理页面顶部，控制 SEO / 内容 / GEO 自动翻译轮询的启停
 */
const AutoTranslateToggle = ({ compact = false }) => {
  const { t } = useTranslation();
  const [enabled, setEnabled] = useState(true);
  const [envDisabled, setEnvDisabled] = useState(false);
  const [disableReason, setDisableReason] = useState('');
  const [loading, setLoading] = useState(false);
  const [mounted, setMounted] = useState(false);

  const fetchStatus = useCallback(async () => {
    try {
      const res = await API.get('/api/admin/auto-translate-status');
      if (res.data.success) {
        const data = res.data.data;
        setEnabled(data.enabled);
        setEnvDisabled(data.env_disabled);
        setDisableReason(data.disable_reason || '');
        setMounted(true);
      }
    } catch (e) {
      // 静默失败，不影响页面使用
      console.warn('[AutoTranslateToggle] Failed to load status:', e);
    }
  }, []);

  useEffect(() => {
    fetchStatus();
  }, [fetchStatus]);

  const handleToggle = async (checked) => {
    setLoading(true);
    try {
      const res = await API.put('/api/admin/auto-translate-status', { enabled: checked });
      if (res.data.success) {
        setEnabled(checked);
        showSuccess(checked ? t('系统自动翻译已开启') : t('系统自动翻译已关闭'));
      } else {
        showError(res.data.message || t('操作失败'));
        // 回滚状态
        fetchStatus();
      }
    } catch (e) {
      showError(e.message || t('操作失败'));
      fetchStatus();
    } finally {
      setLoading(false);
    }
  };

  // 未加载完成时不渲染
  if (!mounted) return null;

  // 紧凑模式：只显示 Switch + 小标签
  if (compact) {
    return (
      <div className="flex items-center gap-2">
        <Switch
          checked={enabled}
          onChange={handleToggle}
          loading={loading}
          disabled={envDisabled}
          checkedText={t('开')}
          uncheckedText={t('关')}
          size="small"
        />
        <Tag
          color={enabled ? 'green' : 'red'}
          size="small"
        >
          {enabled ? t('自动翻译运行中') : t('自动翻译已暂停')}
        </Tag>
        {envDisabled && (
          <Tag color="grey" size="small" title={disableReason}>
            {t('环境变量禁用')}
          </Tag>
        )}
      </div>
    );
  }

  // 完整模式：带说明横幅
  return (
    <Banner
      type={enabled ? 'info' : 'warning'}
      icon={enabled ? <IconPlay /> : <IconPause />}
      title={
        <Space spacing="tight">
          <Text strong>{t('系统自动翻译')}</Text>
          <Switch
            checked={enabled}
            onChange={handleToggle}
            loading={loading}
            disabled={envDisabled}
            style={{ marginLeft: 8 }}
            checkedText={t('开')}
            uncheckedText={t('关')}
          />
          {!enabled && (
            <Tag color="red" size="small">
              {envDisabled ? t('环境变量强制禁用') : t('已手动暂停')}
            </Tag>
          )}
        </Space>
      }
      description={t(
        enabled
          ? '自动翻译正在运行中（每10分钟扫描一次），系统会自动翻译未完成的 SEO / 内容 / GEO 多语言。'
          : envDisabled
            ? disableReason || '自动翻译已被环境变量 DISABLE_SEO_AUTO_TRANSLATE=true 强制禁用'
            : '自动翻译已暂停，新内容不会被自动翻译。可以手动触发批量翻译。'
      )}
      style={{ marginBottom: 16, borderRadius: 8 }}
    />
  );
};

export default AutoTranslateToggle;
