/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Row,
  Col,
  Tag,
  Space,
  Spin,
  Typography,
  SideSheet,
  Pagination,
  Avatar,
  Empty,
  TextArea,
  Progress,
  Collapse,
} from '@douyinfe/semi-ui';
import {
  IconSave,
  IconClose,
  IconSearch,
  IconRefresh,
  IconEdit,
  IconBookStroked,
  IconLanguage,
  IconTickCircle,
  IconAlertTriangle,
  IconBolt,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';

const { Text, Title } = Typography;

const LANGUAGES = [
  { code: 'zh', label: '中文' },
  { code: 'en', label: 'English' },
  { code: 'fr', label: 'Français' },
  { code: 'ru', label: 'Русский' },
  { code: 'ja', label: '日本語' },
  { code: 'vi', label: 'Tiếng Việt' },
  { code: 'ko', label: '한국어' },
  { code: 'es', label: 'Español' },
  { code: 'de', label: 'Deutsch' },
  { code: 'it', label: 'Italiano' },
  { code: 'pt', label: 'Português' },
  { code: 'ar', label: 'العربية' },
];
const DEFAULT_LANG = 'zh';

const emptySeoI18n = () => {
  const obj = {};
  LANGUAGES.forEach((lang) => {
    obj[lang.code] = { seo_title: '', seo_description: '', seo_keywords: '' };
  });
  return obj;
};

const SEOEditModal = ({ visible, onCancel, articleId, refresh }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const formApiRef = useRef(null);
  const [data, setData] = useState({
    title: '',
    category_name: '',
    seo_title: '',
    seo_description: '',
    seo_keywords: '',
  });
  const [activeLang, setActiveLang] = useState(DEFAULT_LANG);
  const [i18nData, setI18nData] = useState(emptySeoI18n());
  const [translating, setTranslating] = useState(false);

  const loadDetail = async () => {
    if (!articleId) return;
    setLoading(true);
    try {
      const res = await API.get(`/api/article/seo/${articleId}`);
      const { success, message, data } = res.data;
      if (success) {
        setData(data);
        let parsed = {};
        try {
          if (data.seo_i18n) parsed = JSON.parse(data.seo_i18n);
        } catch (e) {}
        const merged = emptySeoI18n();
        Object.keys(parsed).forEach((code) => {
          if (merged[code]) merged[code] = { ...merged[code], ...parsed[code] };
        });
        setI18nData(merged);
        setActiveLang(DEFAULT_LANG);
        formApiRef.current?.setValues({
          seo_title: data.seo_title || '',
          seo_description: data.seo_description || '',
          seo_keywords: data.seo_keywords || '',
        });
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  useEffect(() => {
    if (visible && articleId) {
      loadDetail();
    }
  }, [visible, articleId]);

  const buildTranslateItems = (values) => {
    const items = [
      { key: 'seo_title', text: values.seo_title || '' },
      { key: 'seo_description', text: values.seo_description || '' },
      { key: 'seo_keywords', text: values.seo_keywords || '' },
    ];
    return items.filter((item) => item.text !== '');
  };

  const handleAutoTranslate = async () => {
    if (!formApiRef.current) return;
    const values = formApiRef.current.getValues();
    const items = buildTranslateItems(values);
    if (items.length === 0) {
      showError('请先填写中文 SEO 内容');
      return;
    }
    const targetLangs = LANGUAGES.filter((l) => l.code !== DEFAULT_LANG).map((l) => l.code);
    setTranslating(true);
    try {
      const res = await API.post('/api/translate/batch', {
        items,
        source_lang: DEFAULT_LANG,
        target_langs: targetLangs,
      });
      const { success, data: result, message } = res.data;
      if (success && result) {
        const updated = { ...i18nData };
        Object.keys(result).forEach((langCode) => {
          if (updated[langCode]) {
            const langResult = result[langCode];
            updated[langCode] = {
              ...updated[langCode],
              seo_title: langResult.seo_title || updated[langCode].seo_title || '',
              seo_description: langResult.seo_description || updated[langCode].seo_description || '',
              seo_keywords: langResult.seo_keywords || updated[langCode].seo_keywords || '',
            };
          }
        });
        setI18nData(updated);
        showSuccess('翻译完成');
      } else {
        showError(message || '翻译失败');
      }
    } catch (err) {
      showError(err.message);
    } finally {
      setTranslating(false);
    }
  };

  const handleRetranslate = async (targetLang) => {
    if (!formApiRef.current) return;
    const values = formApiRef.current.getValues();
    const items = buildTranslateItems(values);
    if (items.length === 0) {
      showError('请先填写中文 SEO 内容');
      return;
    }
    setTranslating(true);
    try {
      const res = await API.post('/api/translate/batch', {
        items,
        source_lang: DEFAULT_LANG,
        target_langs: [targetLang],
      });
      const { success, data: result, message } = res.data;
      if (success && result && result[targetLang]) {
        const langResult = result[targetLang];
        setI18nData((prev) => ({
          ...prev,
          [targetLang]: {
            ...prev[targetLang],
            seo_title: langResult.seo_title || prev[targetLang]?.seo_title || '',
            seo_description: langResult.seo_description || prev[targetLang]?.seo_description || '',
            seo_keywords: langResult.seo_keywords || prev[targetLang]?.seo_keywords || '',
          },
        }));
        showSuccess('翻译完成');
      } else {
        showError(message || '翻译失败');
      }
    } catch (err) {
      showError(err.message);
    } finally {
      setTranslating(false);
    }
  };

  const updateI18nField = (langCode, field, value) => {
    setI18nData((prev) => ({
      ...prev,
      [langCode]: { ...prev[langCode], [field]: value },
    }));
  };

  const submit = async (values) => {
    setSaving(true);
    try {
      const res = await API.put(`/api/article/seo/${articleId}`, {
        id: articleId,
        seo_title: values.seo_title || '',
        seo_description: values.seo_description || '',
        seo_keywords: values.seo_keywords || '',
        seo_i18n: JSON.stringify(i18nData),
      });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('SEO 内容更新成功'));
        refresh();
        onCancel();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setSaving(false);
  };

  const isDefault = activeLang === DEFAULT_LANG;

  return (
    <SideSheet
      title={
        <div className='flex items-center justify-between' style={{ paddingRight: 32 }}>
          <Space>
            <Tag color='blue' shape='circle'>{t('编辑')}</Tag>
            <Title heading={4} className='m-0'>{t('编辑 SEO 内容')}</Title>
          </Space>
          <Button
            icon={<IconLanguage />}
            type='tertiary'
            size='small'
            loading={translating}
            onClick={handleAutoTranslate}
          >
            {t('自动翻译')}
          </Button>
        </div>
      }
      bodyStyle={{ padding: '0' }}
      visible={visible}
      width={700}
      footer={
        <div className='flex justify-end bg-white'>
          <Space>
            <Button
              theme='solid'
              onClick={() => formApiRef.current?.submitForm()}
              icon={<IconSave />}
              loading={saving}
            >
              {t('保存')}
            </Button>
            <Button
              theme='light'
              type='primary'
              onClick={onCancel}
              icon={<IconClose />}
            >
              {t('取消')}
            </Button>
          </Space>
        </div>
      }
      closeIcon={null}
      onCancel={onCancel}
    >
      <Spin spinning={loading}>
        <Form
          initValues={data}
          getFormApi={(api) => (formApiRef.current = api)}
          onSubmit={submit}
        >
          {() => (
            <div className='p-4'>
              <Card className='!rounded-2xl shadow-sm border-0 mb-4'>
                <div className='flex items-center mb-3'>
                  <Avatar size='small' color='blue' className='mr-2 shadow-md'>
                    <IconBookStroked size={16} />
                  </Avatar>
                  <div>
                    <Text className='text-lg font-medium'>
                      {data.title || t('文章信息')}
                    </Text>
                    {data.category_name && (
                      <div className='text-xs text-gray-600'>
                        {t('分类')}: {data.category_name}
                      </div>
                    )}
                  </div>
                </div>

                {/* 自定义语言 Tab 切换 */}
                <div
                  style={{
                    display: 'flex',
                    flexWrap: 'wrap',
                    gap: 2,
                    borderBottom: '1px solid var(--semi-color-border)',
                    marginBottom: 16,
                  }}
                >
                  {LANGUAGES.map((lang) => {
                    const active = activeLang === lang.code;
                    return (
                      <button
                        key={lang.code}
                        type='button'
                        onClick={() => setActiveLang(lang.code)}
                        style={{
                          padding: '8px 14px',
                          border: 'none',
                          background: 'none',
                          cursor: 'pointer',
                          borderBottom: active ? '2px solid var(--semi-color-primary)' : '2px solid transparent',
                          color: active ? 'var(--semi-color-primary)' : 'var(--semi-color-text-2)',
                          fontWeight: active ? 600 : 400,
                          fontSize: 14,
                          transition: 'all 0.2s',
                          marginBottom: -1,
                        }}
                      >
                        {lang.label}
                      </button>
                    );
                  })}
                </div>

                {/* 默认语言字段 */}
                <div style={{ display: isDefault ? 'block' : 'none' }}>
                  <Row gutter={12}>
                    <Col span={24}>
                      <Form.Input
                        field='seo_title'
                        label={t('SEO 标题')}
                        placeholder={t('50-60 字符，包含主关键词')}
                        showClear
                      />
                    </Col>
                    <Col span={24}>
                      <Form.TextArea
                        field='seo_description'
                        label={t('SEO 描述')}
                        placeholder={t('150-160 字符，吸引点击')}
                        rows={3}
                        showClear
                      />
                    </Col>
                    <Col span={24}>
                      <Form.TextArea
                        field='seo_keywords'
                        label={t('SEO 关键词')}
                        placeholder={t('8-12 个关键词，逗号分隔')}
                        rows={2}
                        showClear
                      />
                    </Col>
                  </Row>
                </div>

                {/* 非默认语言字段 */}
                {!isDefault && (
                  <div>
                    <div style={{ marginBottom: 12 }}>
                      <Button
                        icon={<IconLanguage />}
                        type='tertiary'
                        size='small'
                        loading={translating}
                        onClick={() => handleRetranslate(activeLang)}
                      >
                        {t('重新翻译')}
                      </Button>
                    </div>
                    <Row gutter={12}>
                      <Col span={24}>
                        <div style={{ marginBottom: 12 }}>
                          <label style={{ display: 'block', marginBottom: 4, fontWeight: 500, color: 'var(--semi-color-text-0)' }}>
                            {t('SEO 标题')}
                          </label>
                          <Input
                            value={i18nData[activeLang]?.seo_title || ''}
                            onChange={(v) => updateI18nField(activeLang, 'seo_title', v)}
                            placeholder={t('输入翻译后的 SEO 标题')}
                          />
                        </div>
                      </Col>
                      <Col span={24}>
                        <div style={{ marginBottom: 12 }}>
                          <label style={{ display: 'block', marginBottom: 4, fontWeight: 500, color: 'var(--semi-color-text-0)' }}>
                            {t('SEO 描述')}
                          </label>
                          <TextArea
                            value={i18nData[activeLang]?.seo_description || ''}
                            onChange={(v) => updateI18nField(activeLang, 'seo_description', v)}
                            placeholder={t('输入翻译后的 SEO 描述')}
                            rows={3}
                          />
                        </div>
                      </Col>
                      <Col span={24}>
                        <div style={{ marginBottom: 12 }}>
                          <label style={{ display: 'block', marginBottom: 4, fontWeight: 500, color: 'var(--semi-color-text-0)' }}>
                            {t('SEO 关键词')}
                          </label>
                          <Input
                            value={i18nData[activeLang]?.seo_keywords || ''}
                            onChange={(v) => updateI18nField(activeLang, 'seo_keywords', v)}
                            placeholder={t('输入翻译后的 SEO 关键词')}
                          />
                        </div>
                      </Col>
                    </Row>
                  </div>
                )}
              </Card>
            </div>
          )}
        </Form>
      </Spin>
    </SideSheet>
  );
};


const ArticleSEOManagement = () => {
  const { t } = useTranslation();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [total, setTotal] = useState(0);
  const [keyword, setKeyword] = useState('');
  const [editingId, setEditingId] = useState(null);
  const [showEdit, setShowEdit] = useState(false);
  const [regenerating, setRegenerating] = useState({});
  const [auditing, setAuditing] = useState({});
  const [auditResult, setAuditResult] = useState(null);
  const [showAudit, setShowAudit] = useState(false);
  const [auditId, setAuditId] = useState(null);
  const [auditHistory, setAuditHistory] = useState([]);
  const [stats, setStats] = useState(null);

  const loadData = useCallback(
    async (page = activePage, size = pageSize, search = keyword) => {
      setLoading(true);
      try {
        let url = `/api/article/seo/list?p=${page}&page_size=${size}`;
        if (search) {
          url += `&keyword=${encodeURIComponent(search)}`;
        }
        const res = await API.get(url);
        const { success, message, data } = res.data;
        if (success) {
          setItems(data.items || []);
          setActivePage(data.page <= 0 ? 1 : data.page);
          setTotal(data.total || 0);
        } else {
          showError(message);
        }
      } catch (error) {
        showError(error.message);
      }
      setLoading(false);
    },
    [activePage, pageSize, keyword]
  );

  const loadStats = useCallback(async () => {
    try {
      const res = await API.get('/api/article/seo/stats');
      if (res.data.success) {
        setStats(res.data.data);
      }
    } catch (error) {
      // 静默失败
    }
  }, []);

  useEffect(() => {
    loadData(1, pageSize, '');
    loadStats();
  }, [pageSize]);

  const handleSearch = () => {
    setActivePage(1);
    loadData(1, pageSize, keyword);
  };

  const handlePageChange = (page) => {
    setActivePage(page);
    loadData(page, pageSize, keyword);
  };

  const handleRegenerate = async (id) => {
    setRegenerating((prev) => ({ ...prev, [id]: true }));
    try {
      const res = await API.post(`/api/article/seo/${id}/regenerate`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('AI 重新生成已启动，3秒后自动刷新'));
        setTimeout(() => {
          loadData(activePage, pageSize, keyword);
        }, 3000);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setRegenerating((prev) => ({ ...prev, [id]: false }));
  };

  const openEdit = (id) => {
    setEditingId(id);
    setShowEdit(true);
  };

  const closeEdit = () => {
    setShowEdit(false);
    setEditingId(null);
  };

  const openAudit = (id) => {
    setAuditId(id);
    setShowAudit(true);
    loadAuditHistory(id);
  };

  const closeAudit = () => {
    setShowAudit(false);
    setAuditResult(null);
    setAuditId(null);
    setAuditHistory([]);
  };

  const loadAuditHistory = async (id) => {
    try {
      const res = await API.get(`/api/article/seo/${id}/audits?limit=10`);
      if (res.data.success) {
        setAuditHistory(res.data.data || []);
      }
    } catch (error) {
      setAuditHistory([]);
    }
  };

  const handleAudit = async (id) => {
    setAuditing((prev) => ({ ...prev, [id]: true }));
    setAuditResult(null);
    try {
      const historyRes = await API.get(`/api/article/seo/${id}/audits?limit=1`);
      if (historyRes.data.success && historyRes.data.data && historyRes.data.data.length > 0) {
        const latest = historyRes.data.data[0];
        setAuditResult({
          overall_score: latest.overall_score,
          categories: JSON.parse(latest.categories || '{}'),
          critical_issues: JSON.parse(latest.critical_issues || '[]'),
          quick_wins: JSON.parse(latest.quick_wins || '[]'),
        });
        openAudit(id);
      } else {
        const res = await API.post(`/api/article/seo/${id}/audit`);
        const { success, data, message } = res.data;
        if (success) {
          setAuditResult(data);
          openAudit(id);
        } else {
          showError(message || t('审计失败'));
        }
      }
    } catch (error) {
      showError(error.message);
    }
    setAuditing((prev) => ({ ...prev, [id]: false }));
  };

  const handleReAudit = async (id) => {
    setAuditing((prev) => ({ ...prev, [id]: true }));
    setAuditResult(null);
    try {
      const res = await API.post(`/api/article/seo/${id}/audit`);
      const { success, data, message } = res.data;
      if (success) {
        setAuditResult(data);
        loadAuditHistory(id);
        loadStats();
        loadData(activePage, pageSize, keyword);
        showSuccess(t('审计完成'));
      } else {
        showError(message || t('审计失败'));
      }
    } catch (error) {
      showError(error.message);
    }
    setAuditing((prev) => ({ ...prev, [id]: false }));
  };

  const truncate = (text, maxLen = 60) => {
    if (!text) return '-';
    return text.length > maxLen ? text.slice(0, maxLen) + '...' : text;
  };

  const renderAuditScore = (score) => {
    if (!score || score <= 0) return <Tag size='small'>{t('未审计')}</Tag>;
    const color = score >= 80 ? 'green' : score >= 60 ? 'orange' : 'red';
    return (
      <Tag color={color} size='small' shape='circle'>
        {score}
      </Tag>
    );
  };

  const hasSEO = (item) => {
    return !!(item.seo_title || item.seo_description || item.seo_keywords);
  };

  return (
    <div className='mt-[60px] px-2'>
      {/* SEO 统计卡片 */}
      {stats && (
        <Row gutter={12} className='mb-4'>
          <Col span={6} xs={24} sm={12} md={6}>
            <Card className='!rounded-2xl shadow-sm border-0' bodyStyle={{ padding: '16px' }}>
              <div className='text-sm text-gray-500 mb-1'>{t('SEO 覆盖率')}</div>
              <div className='text-2xl font-bold' style={{ color: stats.seo_coverage >= 80 ? '#52c41a' : stats.seo_coverage >= 50 ? '#faad14' : '#f5222d' }}>
                {stats.seo_coverage}%
              </div>
              <div className='text-xs text-gray-400'>
                {stats.with_seo} / {stats.total_articles}
              </div>
            </Card>
          </Col>
          <Col span={6} xs={24} sm={12} md={6}>
            <Card className='!rounded-2xl shadow-sm border-0' bodyStyle={{ padding: '16px' }}>
              <div className='text-sm text-gray-500 mb-1'>{t('审计覆盖率')}</div>
              <div className='text-2xl font-bold' style={{ color: stats.audit_coverage >= 80 ? '#52c41a' : stats.audit_coverage >= 50 ? '#faad14' : '#f5222d' }}>
                {stats.audit_coverage}%
              </div>
              <div className='text-xs text-gray-400'>
                {stats.with_audit} / {stats.total_articles}
              </div>
            </Card>
          </Col>
          <Col span={6} xs={24} sm={12} md={6}>
            <Card className='!rounded-2xl shadow-sm border-0' bodyStyle={{ padding: '16px' }}>
              <div className='text-sm text-gray-500 mb-1'>{t('平均审计分')}</div>
              <div className='text-2xl font-bold' style={{ color: stats.average_score >= 80 ? '#52c41a' : stats.average_score >= 60 ? '#faad14' : '#f5222d' }}>
                {stats.average_score}
              </div>
              <div className='text-xs text-gray-400'>
                {t('满分 100')}
              </div>
            </Card>
          </Col>
          <Col span={6} xs={24} sm={12} md={6}>
            <Card className='!rounded-2xl shadow-sm border-0' bodyStyle={{ padding: '16px' }}>
              <div className='text-sm text-gray-500 mb-1'>{t('分数分布')}</div>
              <div className='flex gap-2 text-xs'>
                {stats.score_distribution?.map((d) => {
                  const labelMap = {
                    excellent: t('优'),
                    good: t('良'),
                    average: t('中'),
                    poor: t('差'),
                  };
                  const colorMap = {
                    excellent: '#52c41a',
                    good: '#95de64',
                    average: '#faad14',
                    poor: '#f5222d',
                  };
                  return (
                    <span key={d.range} style={{ color: colorMap[d.range] || '#666' }}>
                      {labelMap[d.range] || d.range}: {d.count}
                    </span>
                  );
                }) || '-'}
              </div>
            </Card>
          </Col>
        </Row>
      )}

      <Card className='!rounded-2xl shadow-sm border-0 mb-4'>
        <div className='flex flex-wrap items-center justify-between gap-3 mb-4'>
          <div className='flex items-center gap-2'>
            <Avatar size='small' color='green'>
              <IconBookStroked size={16} />
            </Avatar>
            <Text className='text-lg font-medium'>{t('SEO 管理')}</Text>
          </div>
          <div className='flex items-center gap-2'>
            <Input
              placeholder={t('搜索文章标题')}
              value={keyword}
              onChange={(v) => setKeyword(v)}
              onEnterPress={handleSearch}
              showClear
              suffix={
                <Button
                  type='tertiary'
                  size='small'
                  icon={<IconSearch />}
                  onClick={handleSearch}
                />
              }
            />
          </div>
        </div>

        <Spin spinning={loading}>
          <div className='overflow-x-auto'>
            <table className='w-full text-sm'>
              <thead>
                <tr
                  className='border-b'
                  style={{ borderColor: 'var(--semi-color-border)' }}
                >
                  <th className='text-left py-2 px-3 font-medium w-16'>
                    {t('ID')}
                  </th>
                  <th className='text-left py-2 px-3 font-medium'>
                    {t('标题')}
                  </th>
                  <th className='text-left py-2 px-3 font-medium'>
                    {t('分类')}
                  </th>
                  <th className='text-left py-2 px-3 font-medium'>
                    {t('SEO 标题')}
                  </th>
                  <th className='text-left py-2 px-3 font-medium'>
                    {t('SEO 描述')}
                  </th>
                  <th className='text-left py-2 px-3 font-medium'>
                    {t('SEO 关键词')}
                  </th>
                  <th className='text-left py-2 px-3 font-medium w-24'>
                    {t('审计评分')}
                  </th>
                  <th className='text-left py-2 px-3 font-medium w-24'>
                    {t('状态')}
                  </th>
                  <th className='text-right py-2 px-3 font-medium w-40'>
                    {t('操作')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr
                    key={item.id}
                    className='border-b hover:bg-gray-50'
                    style={{ borderColor: 'var(--semi-color-border)' }}
                  >
                    <td className='py-2 px-3'>{item.id}</td>
                    <td className='py-2 px-3 font-medium'>
                      <div className='flex items-center gap-1'>
                        {item.title}
                        {hasSEO(item) && (
                          <Tag color='green' size='small'>
                            SEO
                          </Tag>
                        )}
                      </div>
                    </td>
                    <td className='py-2 px-3'>
                      {item.category_name || '-'}
                    </td>
                    <td className='py-2 px-3 text-gray-600'>
                      {truncate(item.seo_title, 40)}
                    </td>
                    <td className='py-2 px-3 text-gray-600'>
                      {truncate(item.seo_description, 50)}
                    </td>
                    <td className='py-2 px-3 text-gray-600'>
                      {truncate(item.seo_keywords, 40)}
                    </td>
                    <td className='py-2 px-3'>
                      {renderAuditScore(item.audit_score)}
                    </td>
                    <td className='py-2 px-3'>
                      <Tag
                        color={item.status === 1 ? 'green' : 'red'}
                        shape='circle'
                        size='small'
                      >
                        {item.status === 1 ? t('启用') : t('禁用')}
                      </Tag>
                    </td>
                    <td className='py-2 px-3 text-right'>
                      <Space>
                        <Button
                          type='tertiary'
                          size='small'
                          icon={<IconEdit />}
                          onClick={() => openEdit(item.id)}
                        >
                          {t('编辑')}
                        </Button>
                        <Button
                          type='tertiary'
                          size='small'
                          icon={<IconRefresh />}
                          loading={regenerating[item.id]}
                          onClick={() => handleRegenerate(item.id)}
                        >
                          {t('重新生成')}
                        </Button>
                        <Button
                          type='tertiary'
                          size='small'
                          icon={<IconSearch />}
                          loading={auditing[item.id]}
                          onClick={() => handleAudit(item.id)}
                        >
                          {t('审计')}
                        </Button>
                      </Space>
                    </td>
                  </tr>
                ))}
                {items.length === 0 && (
                  <tr>
                    <td
                      colSpan={9}
                      className='py-8 text-center text-gray-400'
                    >
                      <Empty description={t('暂无数据')} />
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {total > 0 && (
            <div className='flex justify-end mt-4'>
              <Pagination
                currentPage={activePage}
                pageSize={pageSize}
                total={total}
                showSizeChanger
                pageSizeOpts={[10, 20, 50, 100]}
                onPageChange={handlePageChange}
                onPageSizeChange={(size) => {
                  setPageSize(size);
                  setActivePage(1);
                }}
              />
            </div>
          )}
        </Spin>
      </Card>

      <SEOEditModal
        visible={showEdit}
        onCancel={closeEdit}
        articleId={editingId}
        refresh={() => loadData(activePage, pageSize, keyword)}
      />

      {/* SEO 审计报告面板 */}
      <SideSheet
        title={
          <Space>
            <Tag color='blue' shape='circle'>{t('审计')}</Tag>
            <Title heading={4} className='m-0'>
              {t('SEO 审计报告')}
            </Title>
          </Space>
        }
        visible={showAudit}
        width={600}
        onCancel={closeAudit}
        closeIcon={null}
      >
        {auditResult ? (
          <div className='p-2'>
            {/* 操作栏 */}
            <div className='flex justify-end gap-2 mb-4'>
              <Button
                type='tertiary'
                size='small'
                icon={<IconRefresh />}
                loading={auditing[auditId]}
                onClick={() => handleReAudit(auditId)}
              >
                {t('重新审计')}
              </Button>
              <Button
                type='tertiary'
                size='small'
                icon={<IconBolt />}
                loading={regenerating[auditId]}
                onClick={() => handleRegenerate(auditId)}
              >
                {t('根据建议重新生成 SEO')}
              </Button>
            </div>

            {/* 总分 */}
            <Card className='!rounded-2xl shadow-sm border-0 mb-4'>
              <div className='flex items-center gap-4'>
                <div className='relative w-20 h-20 flex items-center justify-center'>
                  <Progress
                    percent={auditResult.overall_score}
                    type='circle'
                    size='small'
                    stroke={auditResult.overall_score >= 80 ? '#52c41a' : auditResult.overall_score >= 60 ? '#faad14' : '#f5222d'}
                  />
                </div>
                <div>
                  <div className='text-2xl font-bold' style={{ color: auditResult.overall_score >= 80 ? '#52c41a' : auditResult.overall_score >= 60 ? '#faad14' : '#f5222d' }}>
                    {auditResult.overall_score} / 100
                  </div>
                  <div className='text-sm text-gray-500'>
                    {auditResult.overall_score >= 80 ? t('优秀') : auditResult.overall_score >= 60 ? t('良好') : t('需改进')}
                  </div>
                </div>
              </div>
            </Card>

            {/* 各维度分数 */}
            <Card className='!rounded-2xl shadow-sm border-0 mb-4' title={t('维度评分')}>
              <div className='space-y-3'>
                {Object.entries(auditResult.categories || {}).map(([key, cat]) => {
                  const labelMap = {
                    completeness: t('完整性'),
                    keyword_quality: t('关键词质量'),
                    title_quality: t('标题质量'),
                    description_quality: t('描述质量'),
                    technical: t('技术规范'),
                  };
                  return (
                    <div key={key}>
                      <div className='flex justify-between text-sm mb-1'>
                        <span>{labelMap[key] || key}</span>
                        <span className='font-medium'>{cat.score}</span>
                      </div>
                      <Progress
                        percent={cat.score}
                        stroke={cat.score >= 80 ? '#52c41a' : cat.score >= 60 ? '#faad14' : '#f5222d'}
                        showInfo={false}
                      />
                    </div>
                  );
                })}
              </div>
            </Card>

            {/* 关键问题 & 快速改进 */}
            {(auditResult.critical_issues?.length > 0 || auditResult.quick_wins?.length > 0) && (
              <Card className='!rounded-2xl shadow-sm border-0 mb-4'>
                {auditResult.critical_issues?.length > 0 && (
                  <div className='mb-4'>
                    <div className='flex items-center gap-2 mb-2'>
                      <IconAlertTriangle size={16} style={{ color: '#f5222d' }} />
                      <Text strong>{t('关键问题')}</Text>
                    </div>
                    <div className='space-y-2'>
                      {auditResult.critical_issues.map((issue, idx) => (
                        <div key={idx} className='flex items-start gap-2 text-sm'>
                          <span className='text-red-500 mt-0.5'>•</span>
                          <span>{issue}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
                {auditResult.quick_wins?.length > 0 && (
                  <div>
                    <div className='flex items-center gap-2 mb-2'>
                      <IconBolt size={16} style={{ color: '#52c41a' }} />
                      <Text strong>{t('快速改进')}</Text>
                    </div>
                    <div className='space-y-2'>
                      {auditResult.quick_wins.map((win, idx) => (
                        <div key={idx} className='flex items-start gap-2 text-sm'>
                          <span className='text-green-500 mt-0.5'>•</span>
                          <span>{win}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </Card>
            )}

            {/* 各维度详情 */}
            <Card className='!rounded-2xl shadow-sm border-0 mb-4' title={t('详细建议')}>
              <Collapse accordion>
                {Object.entries(auditResult.categories || {}).map(([key, cat]) => {
                  const labelMap = {
                    completeness: t('完整性'),
                    keyword_quality: t('关键词质量'),
                    title_quality: t('标题质量'),
                    description_quality: t('描述质量'),
                    technical: t('技术规范'),
                  };
                  const hasContent = (cat.issues?.length > 0) || (cat.suggestions?.length > 0);
                  if (!hasContent) return null;
                  return (
                    <Collapse.Panel
                      key={key}
                      header={
                        <span>
                          {labelMap[key] || key}
                          <Tag
                            size='small'
                            color={cat.score >= 80 ? 'green' : cat.score >= 60 ? 'orange' : 'red'}
                            style={{ marginLeft: 8 }}
                          >
                            {cat.score}
                          </Tag>
                        </span>
                      }
                    >
                      {cat.issues?.length > 0 && (
                        <div className='mb-3'>
                          <Text strong type='danger'>{t('问题')}</Text>
                          <ul className='list-disc pl-5 mt-1 space-y-1 text-sm'>
                            {cat.issues.map((issue, idx) => (
                              <li key={idx}>{issue}</li>
                            ))}
                          </ul>
                        </div>
                      )}
                      {cat.suggestions?.length > 0 && (
                        <div>
                          <Text strong type='success'>{t('建议')}</Text>
                          <ul className='list-disc pl-5 mt-1 space-y-1 text-sm'>
                            {cat.suggestions.map((s, idx) => (
                              <li key={idx}>{s}</li>
                            ))}
                          </ul>
                        </div>
                      )}
                    </Collapse.Panel>
                  );
                })}
              </Collapse>
            </Card>

            {/* 审计历史 */}
            {auditHistory.length > 0 && (
              <Card className='!rounded-2xl shadow-sm border-0 mb-4' title={t('审计历史')}>
                <div className='space-y-2'>
                  {auditHistory.map((h) => (
                    <div key={h.id} className='flex items-center justify-between text-sm py-1 border-b' style={{ borderColor: 'var(--semi-color-border)' }}>
                      <div className='flex items-center gap-2'>
                        <span
                          className='w-2 h-2 rounded-full inline-block'
                          style={{
                            background: h.overall_score >= 80 ? '#52c41a' : h.overall_score >= 60 ? '#faad14' : '#f5222d',
                          }}
                        />
                        <span>{new Date(h.created_at * 1000).toLocaleString()}</span>
                      </div>
                      <Tag
                        size='small'
                        color={h.overall_score >= 80 ? 'green' : h.overall_score >= 60 ? 'orange' : 'red'}
                      >
                        {h.overall_score}
                      </Tag>
                    </div>
                  ))}
                </div>
              </Card>
            )}
          </div>
        ) : (
          <div className='flex items-center justify-center h-64'>
            <Spin size='large' tip={t('AI 审计中...')} />
          </div>
        )}
      </SideSheet>
    </div>
  );
};

export default ArticleSEOManagement;
