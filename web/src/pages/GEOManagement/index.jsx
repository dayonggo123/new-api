/*
Copyright (C) 2025 QuantumNous
*/

import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  Button,
  Card,
  Input,
  Row,
  Col,
  Tag,
  Space,
  Spin,
  Typography,
  SideSheet,
  Pagination,
  Tabs,
  Popconfirm,
  Empty,
  Table,
} from '@douyinfe/semi-ui';
import {
  IconSearch,
  IconRefresh,
  IconLanguage,
  IconGlobeStroke,
  IconCheckCircleStroked,
  IconAlertTriangle,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import { openViewGeoBlocksModal } from '../../components/modals/ViewGeoBlocksModal';

const { Text, Title } = Typography;
const { TabPane } = Tabs;

const GEO_TARGET_LANGS = ['en', 'fr', 'ru', 'ja', 'vi', 'ko', 'es', 'de', 'pt', 'it', 'ar'];

const getGeoProgress = (record) => {
  if (!record.geo_blocks || record.geo_blocks === '') return { completed: 0, total: 0, label: '0/0' };
  let i18nMap = {};
  try {
    if (record.geo_blocks_i18n) i18nMap = JSON.parse(record.geo_blocks_i18n);
  } catch (e) { /* ignore */ }
  const completed = Object.keys(i18nMap).filter(k => i18nMap[k] && String(i18nMap[k]).trim() !== '').length;
  return { completed, total: GEO_TARGET_LANGS.length, label: `${completed}/${GEO_TARGET_LANGS.length}` };
};

const GEOManagement = () => {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('prompt');

  // Prompts
  const [prompts, setPrompts] = useState([]);
  const [promptTotal, setPromptTotal] = useState(0);
  const [promptPage, setPromptPage] = useState(1);
  const [promptSize, setPromptSize] = useState(ITEMS_PER_PAGE);
  const [promptKeyword, setPromptKeyword] = useState('');
  const [promptLoading, setPromptLoading] = useState(false);
  const [promptSelectedKeys, setPromptSelectedKeys] = useState([]);
  const [generatingPrompts, setGeneratingPrompts] = useState({});

  // Articles
  const [articles, setArticles] = useState([]);
  const [articleTotal, setArticleTotal] = useState(0);
  const [articlePage, setArticlePage] = useState(1);
  const [articleSize, setArticleSize] = useState(ITEMS_PER_PAGE);
  const [articleKeyword, setArticleKeyword] = useState('');
  const [articleLoading, setArticleLoading] = useState(false);
  const [articleSelectedKeys, setArticleSelectedKeys] = useState([]);
  const [generatingArticles, setGeneratingArticles] = useState({});

  const pollRef = useRef(null);

  // ========== Load Data ==========
  const loadPrompts = useCallback(async (page, size, kw) => {
    setPromptLoading(true);
    try {
      const res = await API.get(`/api/prompt/?p=${page}&page_size=${size}&keyword=${encodeURIComponent(kw || '')}`);
      const { success, data } = res.data;
      if (success && data) {
        setPrompts(data.items || []);
        setPromptTotal(data.total || 0);
      }
    } catch (error) {
      showError(error.message);
    }
    setPromptLoading(false);
  }, []);

  const loadArticles = useCallback(async (page, size, kw) => {
    setArticleLoading(true);
    try {
      const res = await API.get(`/api/admin/articles?p=${page}&page_size=${size}&keyword=${encodeURIComponent(kw || '')}`);
      const { success, data } = res.data;
      if (success && data) {
        setArticles(data.items || []);
        setArticleTotal(data.total || 0);
      }
    } catch (error) {
      showError(error.message);
    }
    setArticleLoading(false);
  }, []);

  useEffect(() => {
    if (activeTab === 'prompt') {
      loadPrompts(promptPage, promptSize, promptKeyword);
    } else {
      loadArticles(articlePage, articleSize, articleKeyword);
    }
  }, [activeTab]);

  // ========== Batch Generate ==========
  const handleBatchGeneratePromptGeo = async () => {
    if (promptSelectedKeys.length === 0) {
      showError(t('请先选择要生成GEO的提示词'));
      return;
    }
    try {
      const res = await API.post('/api/admin/prompts/geo-blocks/batch', { ids: promptSelectedKeys.slice(0, 50) });
      if (res.data.success) {
        showSuccess(t('已启动批量GEO生成任务'));
        setPromptSelectedKeys([]);
      } else {
        showError(res.data.message || t('批量生成失败'));
      }
    } catch (error) {
      showError(error.message);
    }
  };

  const handleBatchGenerateArticleGeo = async () => {
    if (articleSelectedKeys.length === 0) {
      showError(t('请先选择要生成GEO的文章'));
      return;
    }
    try {
      const res = await API.post('/api/admin/articles/geo-blocks/batch', { ids: articleSelectedKeys.slice(0, 50) });
      if (res.data.success) {
        showSuccess(t('已启动批量GEO生成任务'));
        setArticleSelectedKeys([]);
      } else {
        showError(res.data.message || t('批量生成失败'));
      }
    } catch (error) {
      showError(error.message);
    }
  };

  // ========== Single Generate ==========
  const handleGeneratePromptGeo = async (id) => {
    setGeneratingPrompts(prev => ({ ...prev, [id]: true }));
    try {
      const res = await API.post(`/api/admin/prompts/${id}/geo-blocks`);
      if (res.data.success) {
        showSuccess(t('GEO生成已启动'));
        setTimeout(() => loadPrompts(promptPage, promptSize, promptKeyword), 3000);
      } else {
        showError(res.data.message || t('生成失败'));
      }
    } catch (error) {
      showError(error.message);
    }
    setGeneratingPrompts(prev => ({ ...prev, [id]: false }));
  };

  const handleGenerateArticleGeo = async (id) => {
    setGeneratingArticles(prev => ({ ...prev, [id]: true }));
    try {
      const res = await API.post(`/api/admin/articles/${id}/geo-blocks`);
      if (res.data.success) {
        showSuccess(t('GEO生成已启动'));
        setTimeout(() => loadArticles(articlePage, articleSize, articleKeyword), 3000);
      } else {
        showError(res.data.message || t('生成失败'));
      }
    } catch (error) {
      showError(error.message);
    }
    setGeneratingArticles(prev => ({ ...prev, [id]: false }));
  };

  // ========== Render Helpers ==========
  const renderGeoStatus = (record) => {
    const hasGeo = record.geo_blocks && record.geo_blocks !== '';
    if (hasGeo) {
      return (
        <Tag color='green' size='small'>
          {t('已生成')}
        </Tag>
      );
    }
    return (
      <Tag color='grey' size='small'>
        {t('未生成')}
      </Tag>
    );
  };

  const renderGeoProgress = (record) => {
    const hasGeo = record.geo_blocks && record.geo_blocks !== '';
    if (!hasGeo) return <Tag color='grey' size='small'>-</Tag>;
    const { completed, total, label } = getGeoProgress(record);
    const isDone = completed >= total && total > 0;
    return (
      <Tag color={isDone ? 'green' : 'blue'} size='small'>
        {label}
      </Tag>
    );
  };

  const renderPromptActions = (record) => {
    const hasGeo = record.geo_blocks && record.geo_blocks !== '';
    return (
      <Space>
        {hasGeo ? (
          <Button
            type='tertiary'
            size='small'
            icon={<IconGlobeStroke />}
            onClick={() => openViewGeoBlocksModal(t, record)}
          >
            {t('查看')}
          </Button>
        ) : (
          <Button
            type='primary'
            size='small'
            loading={generatingPrompts[record.id]}
            onClick={() => handleGeneratePromptGeo(record.id)}
          >
            {t('生成GEO')}
          </Button>
        )}
      </Space>
    );
  };

  const renderArticleActions = (record) => {
    const hasGeo = record.geo_blocks && record.geo_blocks !== '';
    return (
      <Space>
        {hasGeo ? (
          <Button
            type='tertiary'
            size='small'
            icon={<IconGlobeStroke />}
            onClick={() => openViewGeoBlocksModal(t, record)}
          >
            {t('查看')}
          </Button>
        ) : (
          <Button
            type='primary'
            size='small'
            loading={generatingArticles[record.id]}
            onClick={() => handleGenerateArticleGeo(record.id)}
          >
            {t('生成GEO')}
          </Button>
        )}
      </Space>
    );
  };

  // ========== Table Columns ==========
  const promptColumns = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    {
      title: t('标题'),
      dataIndex: 'title',
      render: (text, record) => (
        <div>
          <Text strong>{text}</Text>
          {record.category_name && (
            <div style={{ fontSize: 12, color: 'var(--semi-color-text-2)' }}>
              {record.category_name}
            </div>
          )}
        </div>
      ),
    },
    {
      title: t('GEO状态'),
      dataIndex: 'geo_blocks',
      width: 100,
      align: 'center',
      render: (_, record) => renderGeoStatus(record),
    },
    {
      title: t('翻译进度'),
      dataIndex: 'geo_progress',
      width: 110,
      align: 'center',
      render: (_, record) => renderGeoProgress(record),
    },
    {
      title: t('翻译进度'),
      dataIndex: 'translation_progress',
      width: 100,
      align: 'center',
      render: (_, record) => {
        const targetLangs = ['en', 'fr', 'ru', 'ja', 'vi', 'ko', 'es', 'de', 'pt', 'it', 'ar'];
        let titleMap = {};
        let contentMap = {};
        try { if (record.title_i18n) titleMap = JSON.parse(record.title_i18n); } catch (e) {}
        try { if (record.i18n) contentMap = JSON.parse(record.i18n); } catch (e) {}
        let completed = 0;
        for (const lang of targetLangs) {
          const hasTitle = titleMap[lang] && String(titleMap[lang]).trim() !== '';
          let hasContent = contentMap[lang] && String(contentMap[lang]).trim() !== '';
          if (lang === 'en' && record.content_en && String(record.content_en).trim() !== '') hasContent = true;
          if (hasTitle && hasContent) completed++;
        }
        const progress = `${completed}/11`;
        const isDone = completed === 11;
        const hasError = !!record.translation_error;
        return <Tag color={hasError ? 'red' : isDone ? 'green' : 'blue'} size='small'>{progress}</Tag>;
      },
    },
    {
      title: t('操作'),
      dataIndex: 'operate',
      width: 120,
      align: 'center',
      render: (_, record) => renderPromptActions(record),
    },
  ];

  const articleColumns = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    {
      title: t('标题'),
      dataIndex: 'title',
      render: (text) => <Text strong>{text}</Text>,
    },
    {
      title: t('GEO状态'),
      dataIndex: 'geo_blocks',
      width: 100,
      align: 'center',
      render: (_, record) => renderGeoStatus(record),
    },
    {
      title: t('GEO翻译'),
      dataIndex: 'geo_progress',
      width: 110,
      align: 'center',
      render: (_, record) => renderGeoProgress(record),
    },
    {
      title: t('内容翻译'),
      dataIndex: 'translation_progress',
      width: 100,
      align: 'center',
      render: (_, record) => {
        const targetLangs = ['en', 'fr', 'ru', 'ja', 'vi', 'ko', 'es', 'de', 'pt', 'it', 'ar'];
        let titleMap = {};
        let contentMap = {};
        try { if (record.title_i18n) titleMap = JSON.parse(record.title_i18n); } catch (e) {}
        try { if (record.i18n) contentMap = JSON.parse(record.i18n); } catch (e) {}
        let completed = 0;
        for (const lang of targetLangs) {
          const hasTitle = titleMap[lang] && String(titleMap[lang]).trim() !== '';
          let hasContent = contentMap[lang] && String(contentMap[lang]).trim() !== '';
          if (lang === 'en' && record.content_en && String(record.content_en).trim() !== '') hasContent = true;
          if (hasTitle && hasContent) completed++;
        }
        const progress = `${completed}/11`;
        const isDone = completed === 11;
        const hasError = !!record.translation_error;
        return <Tag color={hasError ? 'red' : isDone ? 'green' : 'blue'} size='small'>{progress}</Tag>;
      },
    },
    {
      title: t('操作'),
      dataIndex: 'operate',
      width: 120,
      align: 'center',
      render: (_, record) => renderArticleActions(record),
    },
  ];

  return (
    <div style={{ padding: '24px' }}>
      <Title heading={3} style={{ marginBottom: 24 }}>
        {t('GEO 管理')}
      </Title>

      <Tabs activeKey={activeTab} onChange={setActiveTab} type='line'>
        <TabPane itemKey='prompt' tab={t('提示词 GEO')}>
          <Card style={{ marginTop: 16 }}>
            <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
              <Col span={8}>
                <Input
                  prefix={<IconSearch />}
                  placeholder={t('搜索标题')}
                  value={promptKeyword}
                  onChange={(v) => setPromptKeyword(v)}
                  onEnterPress={() => loadPrompts(1, promptSize, promptKeyword)}
                />
              </Col>
              <Col span={16} style={{ textAlign: 'right' }}>
                <Space>
                  <Button
                    type='primary'
                    icon={<IconGlobeStroke />}
                    onClick={handleBatchGeneratePromptGeo}
                    disabled={promptSelectedKeys.length === 0}
                  >
                    {t('批量生成GEO')}
                  </Button>
                  <Button icon={<IconRefresh />} onClick={() => loadPrompts(promptPage, promptSize, promptKeyword)}>
                    {t('刷新')}
                  </Button>
                </Space>
              </Col>
            </Row>

            <Table
              loading={promptLoading}
              columns={promptColumns}
              dataSource={prompts}
              pagination={false}
              rowKey='id'
              rowSelection={useMemo(() => ({
                selectedRowKeys: promptSelectedKeys,
                onChange: (keys) => setPromptSelectedKeys(keys),
              }), [promptSelectedKeys])}
              empty={<Empty description={t('暂无数据')} />}
            />

            <div style={{ marginTop: 16, display: 'flex', justifyContent: 'flex-end' }}>
              <Pagination
                total={promptTotal}
                pageSize={promptSize}
                currentPage={promptPage}
                onPageChange={(p) => {
                  setPromptPage(p);
                  loadPrompts(p, promptSize, promptKeyword);
                }}
                onPageSizeChange={(s) => {
                  setPromptSize(s);
                  setPromptPage(1);
                  loadPrompts(1, s, promptKeyword);
                }}
              />
            </div>
          </Card>
        </TabPane>

        <TabPane itemKey='article' tab={t('文章 GEO')}>
          <Card style={{ marginTop: 16 }}>
            <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
              <Col span={8}>
                <Input
                  prefix={<IconSearch />}
                  placeholder={t('搜索标题')}
                  value={articleKeyword}
                  onChange={(v) => setArticleKeyword(v)}
                  onEnterPress={() => loadArticles(1, articleSize, articleKeyword)}
                />
              </Col>
              <Col span={16} style={{ textAlign: 'right' }}>
                <Space>
                  <Button
                    type='primary'
                    icon={<IconGlobeStroke />}
                    onClick={handleBatchGenerateArticleGeo}
                    disabled={articleSelectedKeys.length === 0}
                  >
                    {t('批量生成GEO')}
                  </Button>
                  <Button icon={<IconRefresh />} onClick={() => loadArticles(articlePage, articleSize, articleKeyword)}>
                    {t('刷新')}
                  </Button>
                </Space>
              </Col>
            </Row>

            <Table
              loading={articleLoading}
              columns={articleColumns}
              dataSource={articles}
              pagination={false}
              rowKey='id'
              rowSelection={useMemo(() => ({
                selectedRowKeys: articleSelectedKeys,
                onChange: (keys) => setArticleSelectedKeys(keys),
              }), [articleSelectedKeys])}
              empty={<Empty description={t('暂无数据')} />}
            />

            <div style={{ marginTop: 16, display: 'flex', justifyContent: 'flex-end' }}>
              <Pagination
                total={articleTotal}
                pageSize={articleSize}
                currentPage={articlePage}
                onPageChange={(p) => {
                  setArticlePage(p);
                  loadArticles(p, articleSize, articleKeyword);
                }}
                onPageSizeChange={(s) => {
                  setArticleSize(s);
                  setArticlePage(1);
                  loadArticles(1, s, articleKeyword);
                }}
              />
            </div>
          </Card>
        </TabPane>
      </Tabs>
    </div>
  );
};

export default GEOManagement;
