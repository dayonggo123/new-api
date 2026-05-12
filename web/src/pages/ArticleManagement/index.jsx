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
  Tabs,
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
  Table,
  Popconfirm,
  Switch,
  Avatar,
  Pagination,
  TextArea,
  Select,
} from '@douyinfe/semi-ui';
import {
  IconSave,
  IconClose,
  IconPlus,
  IconBookStroked,
  IconEdit,
  IconLanguage,
  IconSearch,
  IconRefresh,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import MarkdownRenderer from '../../components/common/markdown/MarkdownRenderer';

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

const emptyI18n = () => {
  const obj = {};
  LANGUAGES.forEach((lang) => {
    obj[lang.code] = { title: '', summary: '', content: '', seo_title: '', seo_description: '', seo_keywords: '' };
  });
  return obj;
};

// ==================== Category Edit Modal ====================

const CategoryEditModal = ({ visible, onCancel, category, refresh }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const formApiRef = useRef(null);
  const isEdit = category?.id !== undefined;

  const getInitValues = () => ({
    name: '',
    description: '',
    icon: '',
    sort_order: 0,
    status: true,
  });

  useEffect(() => {
    if (formApiRef.current) {
      if (isEdit) {
        formApiRef.current.setValues({
          ...category,
          status: category.status === 1,
        });
      } else {
        formApiRef.current.setValues(getInitValues());
      }
    }
  }, [category?.id, visible]);

  const submit = async (values) => {
    setLoading(true);
    const payload = {
      ...values,
      status: values.status ? 1 : 2,
      sort_order: parseInt(values.sort_order) || 0,
    };
    try {
      let res;
      if (isEdit) {
        res = await API.put(`/api/admin/article-categories/${category.id}`, payload);
      } else {
        res = await API.post('/api/admin/article-categories', payload);
      }
      const { success, message } = res.data;
      if (success) {
        showSuccess(isEdit ? t('分类更新成功！') : t('分类创建成功！'));
        refresh();
        onCancel();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  return (
    <SideSheet
      title={
        <Space>
          {isEdit ? (
            <Tag color='blue' shape='circle'>{t('更新')}</Tag>
          ) : (
            <Tag color='green' shape='circle'>{t('新建')}</Tag>
          )}
          <Title heading={4} className='m-0'>
            {isEdit ? t('更新分类信息') : t('创建新的分类')}
          </Title>
        </Space>
      }
      bodyStyle={{ padding: '0' }}
      visible={visible}
      width={500}
      footer={
        <div className='flex justify-end bg-white'>
          <Space>
            <Button theme='solid' onClick={() => formApiRef.current?.submitForm()} icon={<IconSave />} loading={loading}>
              {t('提交')}
            </Button>
            <Button theme='light' type='primary' onClick={onCancel} icon={<IconClose />}>
              {t('取消')}
            </Button>
          </Space>
        </div>
      }
      closeIcon={null}
      onCancel={onCancel}
    >
      <Spin spinning={loading}>
        <Form initValues={getInitValues()} getFormApi={(api) => (formApiRef.current = api)} onSubmit={submit}>
          {() => (
            <div className='p-4'>
              <Card className='!rounded-2xl shadow-sm border-0'>
                <Row gutter={12}>
                  <Col span={24}>
                    <Form.Input field='name' label={t('名称')} placeholder={t('请输入分类名称')} rules={[{ required: true, message: t('请输入分类名称') }]} showClear />
                  </Col>
                  <Col span={24}>
                    <Form.TextArea field='description' label={t('描述')} placeholder={t('请输入分类描述')} rows={2} />
                  </Col>
                  <Col span={24}>
                    <Form.Input field='icon' label={t('图标')} placeholder={t('请输入图标名称或 URL')} />
                  </Col>
                  <Col span={12}>
                    <Form.InputNumber field='sort_order' label={t('排序')} placeholder={t('请输入排序值')} min={0} />
                  </Col>
                  <Col span={12}>
                    <div className='flex items-center h-full pt-6'>
                      <Form.Switch field='status' label={t('状态')} checkedText={t('启用')} uncheckedText={t('禁用')} />
                    </div>
                  </Col>
                </Row>
              </Card>
            </div>
          )}
        </Form>
      </Spin>
    </SideSheet>
  );
};

// ==================== Article Edit Modal ====================

const EditArticleModal = ({ visible, onCancel, article, refresh, categories }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [activeTab, setActiveTab] = useState('basic');
  const [activeLang, setActiveLang] = useState(DEFAULT_LANG);
  const [i18nData, setI18nData] = useState(emptyI18n());
  const [translating, setTranslating] = useState(false);
  const [previewContent, setPreviewContent] = useState('');
  const formApiRef = useRef(null);
  const isEdit = article?.id !== undefined;

  const categoryOptions = categories.map((c) => ({ label: c.name, value: c.id }));

  const loadDetail = async () => {
    if (!article?.id) return;
    setLoading(true);
    try {
      const res = await API.get(`/api/admin/articles/${article.id}`);
      const { success, message, data } = res.data;
      if (success && data) {
        const values = {
          ...data,
          status: data.status === 1,
          is_featured: data.is_featured === true || data.is_featured === 1,
        };
        formApiRef.current?.setValues(values);
        setPreviewContent(data.content || '');
        let parsed = {};
        try {
          if (data.i18n) parsed = JSON.parse(data.i18n);
        } catch (e) {}
        const merged = emptyI18n();
        Object.keys(parsed).forEach((code) => {
          if (merged[code]) merged[code] = { ...merged[code], ...parsed[code] };
        });
        setI18nData(merged);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  useEffect(() => {
    if (visible && article?.id) {
      loadDetail();
    } else if (visible && !article?.id) {
      // New article
      formApiRef.current?.setValues({
        title: '',
        slug: '',
        category_id: categories[0]?.id || 0,
        author: '',
        cover_image_url: '',
        tags: '',
        status: true,
        is_featured: false,
        summary: '',
        content: '',
        seo_title: '',
        seo_description: '',
        seo_keywords: '',
      });
      setPreviewContent('');
      setI18nData(emptyI18n());
      setActiveTab('basic');
      setActiveLang(DEFAULT_LANG);
    }
  }, [visible, article?.id]);

  const buildTranslateItems = (values) => {
    const items = [
      { key: 'title', text: values.title || '' },
      { key: 'summary', text: values.summary || '' },
      { key: 'content', text: values.content || '' },
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
      showError('请先填写中文内容');
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
              title: langResult.title || updated[langCode].title || '',
              summary: langResult.summary || updated[langCode].summary || '',
              content: langResult.content || updated[langCode].content || '',
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

  const handleRetranslateLang = async (targetLang) => {
    if (!formApiRef.current) return;
    const values = formApiRef.current.getValues();
    const items = buildTranslateItems(values);
    if (items.length === 0) {
      showError('请先填写中文内容');
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
            title: langResult.title || prev[targetLang]?.title || '',
            summary: langResult.summary || prev[targetLang]?.summary || '',
            content: langResult.content || prev[targetLang]?.content || '',
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

  const handleRegenerateSEO = async () => {
    if (!isEdit) {
      showError('请先保存文章');
      return;
    }
    try {
      const res = await API.post(`/api/article/seo/${article.id}/regenerate`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(message || 'AI 生成任务已启动');
      } else {
        showError(message);
      }
    } catch (err) {
      showError(err.message);
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
      const payload = {
        ...values,
        status: values.status ? 1 : 2,
        is_featured: values.is_featured ? true : false,
        category_id: parseInt(values.category_id) || 0,
        i18n: JSON.stringify(i18nData),
      };
      let res;
      if (isEdit) {
        res = await API.put(`/api/admin/articles/${article.id}`, { ...payload, id: article.id });
      } else {
        res = await API.post('/api/admin/articles', payload);
      }
      const { success, message } = res.data;
      if (success) {
        showSuccess(isEdit ? t('文章更新成功') : t('文章创建成功'));
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

  const tabButtons = [
    { key: 'basic', label: t('基本信息') },
    { key: 'content', label: t('正文内容') },
    { key: 'seo', label: 'SEO' },
    { key: 'i18n', label: t('多语言') },
  ];

  return (
    <SideSheet
      title={
        <div className='flex items-center justify-between' style={{ paddingRight: 32 }}>
          <Space>
            <Tag color='blue' shape='circle'>{isEdit ? t('编辑') : t('新建')}</Tag>
            <Title heading={4} className='m-0'>{isEdit ? t('编辑文章') : t('新建文章')}</Title>
          </Space>
          {activeTab === 'i18n' && (
            <Button icon={<IconLanguage />} type='tertiary' size='small' loading={translating} onClick={handleAutoTranslate}>
              {t('自动翻译')}
            </Button>
          )}
        </div>
      }
      bodyStyle={{ padding: '0' }}
      visible={visible}
      width={800}
      footer={
        <div className='flex justify-end bg-white'>
          <Space>
            <Button theme='solid' onClick={() => formApiRef.current?.submitForm()} loading={saving}>
              {t('提交')}
            </Button>
            <Button theme='light' onClick={onCancel}>
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
          getFormApi={(api) => {
            formApiRef.current = api;
            api.subscribeValue('content', (val) => setPreviewContent(val || ''));
          }}
          onSubmit={submit}
        >
          {({ formState, formApi }) => (
            <div className='p-4'>
              {/* Tab Buttons */}
              <div className='flex gap-2 mb-4'>
                {tabButtons.map((tab) => (
                  <Button
                    key={tab.key}
                    type={activeTab === tab.key ? 'primary' : 'tertiary'}
                    size='small'
                    onClick={() => setActiveTab(tab.key)}
                  >
                    {tab.label}
                  </Button>
                ))}
              </div>

              {/* Basic Info Tab */}
              <div style={{ display: activeTab === 'basic' ? 'block' : 'none' }}>
                <Card className='!rounded-2xl shadow-sm border-0'>
                  <Row gutter={12}>
                    <Col span={24}>
                      <Form.Input field='title' label={t('标题')} placeholder={t('请输入文章标题')} rules={[{ required: true, message: t('标题不能为空') }]} showClear />
                    </Col>
                    <Col span={12}>
                      <Form.Input field='slug' label={t('Slug')} placeholder={t('URL 友好标识，留空自动生成')} />
                    </Col>
                    <Col span={12}>
                      <Form.Select field='category_id' label={t('分类')} optionList={categoryOptions} />
                    </Col>
                    <Col span={12}>
                      <Form.Input field='author' label={t('作者')} placeholder={t('作者名称')} />
                    </Col>
                    <Col span={12}>
                      <Form.Input field='cover_image_url' label={t('封面图 URL')} placeholder={t('封面图片地址')} />
                    </Col>
                    <Col span={24}>
                      <Form.Input field='tags' label={t('标签')} placeholder={t('JSON 数组格式，如 ["tag1", "tag2"]')} />
                    </Col>
                    <Col span={8}>
                      <div className='flex items-center h-full pt-6'>
                        <Form.Switch field='status' label={t('状态')} checkedText={t('启用')} uncheckedText={t('禁用')} />
                      </div>
                    </Col>
                    <Col span={8}>
                      <div className='flex items-center h-full pt-6'>
                        <Form.Switch field='is_featured' label={t('精选')} checkedText={t('是')} uncheckedText={t('否')} />
                      </div>
                    </Col>
                    <Col span={24}>
                      <Form.TextArea field='summary' label={t('摘要')} placeholder={t('文章摘要，用于列表展示和 SEO 描述备选')} rows={3} />
                    </Col>
                  </Row>
                </Card>
              </div>

              {/* Content Tab */}
              <div style={{ display: activeTab === 'content' ? 'block' : 'none' }}>
                <Card className='!rounded-2xl shadow-sm border-0'>
                  <div style={{ display: 'flex', gap: 16 }}>
                    <div style={{ flex: 1 }}>
                      <Form.TextArea
                        field='content'
                        label={t('正文内容 (Markdown)')}
                        placeholder={t('支持 Markdown 格式')}
                        style={{ fontFamily: 'monospace', minHeight: 500 }}
                        rows={20}
                        rules={[{ required: true, message: t('内容不能为空') }]}
                      />
                    </div>
                    <div style={{ flex: 1, border: '1px solid var(--semi-color-border)', borderRadius: 8, padding: 16, overflow: 'auto', maxHeight: 540 }}>
                      <Text type='tertiary' size='small' className='mb-2 block'>{t('实时预览')}</Text>
                      <MarkdownRenderer content={previewContent || ''} />
                    </div>
                  </div>
                </Card>
              </div>

              {/* SEO Tab */}
              <div style={{ display: activeTab === 'seo' ? 'block' : 'none' }}>
                <Card className='!rounded-2xl shadow-sm border-0'>
                  <Row gutter={12}>
                    <Col span={24}>
                      <div className='flex justify-end mb-2'>
                        <Button type='tertiary' size='small' onClick={handleRegenerateSEO}>
                          {t('AI 生成 SEO')}
                        </Button>
                      </div>
                    </Col>
                    <Col span={24}>
                      <Form.Input field='seo_title' label={t('SEO 标题')} placeholder={t('50-60 字符，包含主关键词')} showClear />
                    </Col>
                    <Col span={24}>
                      <Form.TextArea field='seo_description' label={t('SEO 描述')} placeholder={t('150-160 字符，吸引点击')} rows={3} />
                    </Col>
                    <Col span={24}>
                      <Form.TextArea field='seo_keywords' label={t('SEO 关键词')} placeholder={t('8-12 个关键词，逗号分隔')} rows={2} />
                    </Col>
                  </Row>
                </Card>
              </div>

              {/* I18n Tab */}
              <div style={{ display: activeTab === 'i18n' ? 'block' : 'none' }}>
                <Card className='!rounded-2xl shadow-sm border-0'>
                  <div className='flex flex-wrap gap-1 mb-4'>
                    {LANGUAGES.map((lang) => (
                      <Button
                        key={lang.code}
                        type={activeLang === lang.code ? 'primary' : 'tertiary'}
                        size='small'
                        onClick={() => setActiveLang(lang.code)}
                      >
                        {lang.label}
                      </Button>
                    ))}
                  </div>
                  {activeLang !== DEFAULT_LANG && (
                    <div className='flex justify-end mb-2'>
                      <Button type='tertiary' size='small' loading={translating} onClick={() => handleRetranslateLang(activeLang)}>
                        {t('重新翻译')}
                      </Button>
                    </div>
                  )}
                  <div style={{ display: activeLang === DEFAULT_LANG ? 'block' : 'none' }}>
                    <Text type='tertiary' size='small' className='mb-2 block'>{t('默认语言（中文）内容请在基本信息、正文内容、SEO 标签页中编辑')}</Text>
                  </div>
                  {activeLang !== DEFAULT_LANG && (
                    <div>
                      <Row gutter={12}>
                        <Col span={24}>
                          <Input value={i18nData[activeLang]?.title || ''} onChange={(v) => updateI18nField(activeLang, 'title', v)} placeholder={t('标题')} />
                        </Col>
                        <Col span={24} className='mt-2'>
                          <TextArea value={i18nData[activeLang]?.summary || ''} onChange={(v) => updateI18nField(activeLang, 'summary', v)} placeholder={t('摘要')} rows={3} />
                        </Col>
                        <Col span={24} className='mt-2'>
                          <TextArea value={i18nData[activeLang]?.content || ''} onChange={(v) => updateI18nField(activeLang, 'content', v)} placeholder={t('正文内容 (Markdown)')} rows={8} style={{ fontFamily: 'monospace' }} />
                        </Col>
                        <Col span={24} className='mt-2'>
                          <Input value={i18nData[activeLang]?.seo_title || ''} onChange={(v) => updateI18nField(activeLang, 'seo_title', v)} placeholder={t('SEO 标题')} />
                        </Col>
                        <Col span={24} className='mt-2'>
                          <TextArea value={i18nData[activeLang]?.seo_description || ''} onChange={(v) => updateI18nField(activeLang, 'seo_description', v)} placeholder={t('SEO 描述')} rows={2} />
                        </Col>
                        <Col span={24} className='mt-2'>
                          <TextArea value={i18nData[activeLang]?.seo_keywords || ''} onChange={(v) => updateI18nField(activeLang, 'seo_keywords', v)} placeholder={t('SEO 关键词')} rows={2} />
                        </Col>
                      </Row>
                    </div>
                  )}
                </Card>
              </div>
            </div>
          )}
        </Form>
      </Spin>
    </SideSheet>
  );
};

// ==================== Main Article Management Page ====================

const ArticleManagement = () => {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('articles');

  // Articles state
  const [articles, setArticles] = useState([]);
  const [articleTotal, setArticleTotal] = useState(0);
  const [articlePage, setArticlePage] = useState(1);
  const [articlePageSize, setArticlePageSize] = useState(ITEMS_PER_PAGE);
  const [articleLoading, setArticleLoading] = useState(false);
  const [articleKeyword, setArticleKeyword] = useState('');
  const [articleCategoryId, setArticleCategoryId] = useState(0);
  const [articleStatus, setArticleStatus] = useState(0);
  const [editingArticle, setEditingArticle] = useState(null);
  const [showEdit, setShowEdit] = useState(false);

  // Categories state
  const [categories, setCategories] = useState([]);
  const [catLoading, setCatLoading] = useState(false);
  const [editingCategory, setEditingCategory] = useState({ id: undefined });
  const [showCatEdit, setShowCatEdit] = useState(false);

  const loadArticles = async (page = articlePage, pageSize = articlePageSize) => {
    setArticleLoading(true);
    try {
      const params = new URLSearchParams();
      params.append('p', page);
      params.append('page_size', pageSize);
      if (articleKeyword) params.append('keyword', articleKeyword);
      if (articleCategoryId > 0) params.append('category_id', articleCategoryId);
      if (articleStatus > 0) params.append('status', articleStatus);
      const res = await API.get(`/api/admin/articles?${params.toString()}`);
      const { success, message, data } = res.data;
      if (success && data) {
        setArticles(data.items || []);
        setArticleTotal(data.total || 0);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setArticleLoading(false);
  };

  const loadCategories = async () => {
    setCatLoading(true);
    try {
      const res = await API.get('/api/admin/article-categories');
      const { success, message, data } = res.data;
      if (success) {
        setCategories(data || []);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setCatLoading(false);
  };

  useEffect(() => {
    loadCategories();
  }, []);

  useEffect(() => {
    loadArticles(1, articlePageSize);
  }, [articleKeyword, articleCategoryId, articleStatus]);

  const handleDeleteArticle = async (id) => {
    try {
      const res = await API.delete(`/api/admin/articles/${id}`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('删除成功'));
        loadArticles();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
  };

  const handleDeleteCategory = async (id) => {
    try {
      const res = await API.delete(`/api/admin/article-categories/${id}`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('操作成功完成！'));
        loadCategories();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
  };

  const getCategoryName = (id) => {
    const cat = categories.find((c) => c.id === id);
    return cat ? cat.name : '-';
  };

  const articleColumns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: t('标题'),
      dataIndex: 'title',
      render: (text, record) => (
        <div>
          <Text strong>{text}</Text>
          {record.is_featured && <Tag color='orange' size='small' className='ml-2'>{t('精选')}</Tag>}
        </div>
      ),
    },
    {
      title: t('分类'),
      dataIndex: 'category_name',
      width: 120,
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 80,
      render: (status) => (
        <Tag color={status === 1 ? 'green' : 'red'} shape='circle'>
          {status === 1 ? t('启用') : t('禁用')}
        </Tag>
      ),
    },
    {
      title: t('浏览量'),
      dataIndex: 'view_count',
      width: 90,
    },
    {
      title: t('创建时间'),
      dataIndex: 'created_time',
      width: 160,
      render: (time) => new Date(time * 1000).toLocaleString(),
    },
    {
      title: t('操作'),
      fixed: 'right',
      width: 150,
      render: (_, record) => (
        <Space>
          <Button type='tertiary' size='small' icon={<IconEdit />} onClick={() => {
            setEditingArticle(record);
            setShowEdit(true);
          }}>
            {t('编辑')}
          </Button>
          <Popconfirm title={t('确定删除此文章吗？')} content={t('此操作不可撤销')} onConfirm={() => handleDeleteArticle(record.id)}>
            <Button type='danger' theme='light' size='small'>
              {t('删除')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className='mt-[60px] px-2'>
      <EditArticleModal
        visible={showEdit}
        onCancel={() => {
          setShowEdit(false);
          setEditingArticle(null);
        }}
        article={editingArticle}
        refresh={loadArticles}
        categories={categories}
      />

      <CategoryEditModal
        visible={showCatEdit}
        onCancel={() => {
          setShowCatEdit(false);
          setEditingCategory({ id: undefined });
        }}
        category={editingCategory}
        refresh={loadCategories}
      />

      <Tabs type='line' activeKey={activeTab} onChange={(key) => setActiveTab(key)}>
        <Tabs.TabPane tab={t('文章管理')} itemKey='articles'>
          <Card className='!rounded-2xl shadow-sm border-0'>
            <div className='flex flex-col md:flex-row justify-between items-start md:items-center gap-3 mb-4'>
              <div className='flex items-center gap-2'>
                <Avatar size='small' color='blue'>
                  <IconBookStroked size={16} />
                </Avatar>
                <Text className='text-lg font-medium'>{t('文章列表')}</Text>
              </div>
              <Button type='primary' size='small' icon={<IconPlus />} onClick={() => {
                setEditingArticle(null);
                setShowEdit(true);
              }}>
                {t('新增文章')}
              </Button>
            </div>

            <div className='flex flex-wrap gap-2 mb-4'>
              <Input
                prefix={<IconSearch />}
                placeholder={t('搜索标题、内容')}
                value={articleKeyword}
                onChange={setArticleKeyword}
                onEnterPress={() => loadArticles(1)}
                showClear
                style={{ width: 200 }}
              />
              <Select
                placeholder={t('全部分类')}
                value={articleCategoryId}
                onChange={(v) => setArticleCategoryId(v || 0)}
                optionList={[{ label: t('全部分类'), value: 0 }, ...categories.map((c) => ({ label: c.name, value: c.id }))]}
                style={{ width: 150 }}
              />
              <Select
                placeholder={t('全部状态')}
                value={articleStatus}
                onChange={(v) => setArticleStatus(v || 0)}
                optionList={[
                  { label: t('全部状态'), value: 0 },
                  { label: t('启用'), value: 1 },
                  { label: t('禁用'), value: 2 },
                ]}
                style={{ width: 120 }}
              />
              <Button type='tertiary' size='small' icon={<IconRefresh />} onClick={() => loadArticles(1)}>
                {t('刷新')}
              </Button>
            </div>

            <Spin spinning={articleLoading}>
              <Table
                columns={articleColumns}
                dataSource={articles}
                pagination={false}
                emptyText={t('暂无数据')}
                size='small'
              />
              <div className='flex justify-end mt-4'>
                <Pagination
                  total={articleTotal}
                  pageSize={articlePageSize}
                  currentPage={articlePage}
                  onPageChange={(page) => {
                    setArticlePage(page);
                    loadArticles(page, articlePageSize);
                  }}
                  showSizeChanger
                  pageSizeOpts={[10, 20, 50, 100]}
                  onShowSizeChange={(current, size) => {
                    setArticlePageSize(size);
                    setArticlePage(1);
                    loadArticles(1, size);
                  }}
                />
              </div>
            </Spin>
          </Card>
        </Tabs.TabPane>

        <Tabs.TabPane tab={t('分类管理')} itemKey='categories'>
          <Card className='!rounded-2xl shadow-sm border-0'>
            <div className='flex justify-between items-center mb-4'>
              <div className='flex items-center gap-2'>
                <Avatar size='small' color='blue'>
                  <IconBookStroked size={16} />
                </Avatar>
                <Text className='text-lg font-medium'>{t('分类列表')}</Text>
              </div>
              <Button type='primary' size='small' icon={<IconPlus />} onClick={() => {
                setEditingCategory({ id: undefined });
                setShowCatEdit(true);
              }}>
                {t('添加分类')}
              </Button>
            </div>
            <Spin spinning={catLoading}>
              <div className='overflow-x-auto'>
                <table className='w-full text-sm'>
                  <thead>
                    <tr className='border-b' style={{ borderColor: 'var(--semi-color-border)' }}>
                      <th className='text-left py-2 px-3 font-medium'>{t('ID')}</th>
                      <th className='text-left py-2 px-3 font-medium'>{t('名称')}</th>
                      <th className='text-left py-2 px-3 font-medium'>{t('描述')}</th>
                      <th className='text-left py-2 px-3 font-medium'>{t('排序')}</th>
                      <th className='text-left py-2 px-3 font-medium'>{t('状态')}</th>
                      <th className='text-right py-2 px-3 font-medium'>{t('操作')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {categories.map((cat) => (
                      <tr key={cat.id} className='border-b hover:bg-gray-50' style={{ borderColor: 'var(--semi-color-border)' }}>
                        <td className='py-2 px-3'>{cat.id}</td>
                        <td className='py-2 px-3 font-medium'>{cat.name}</td>
                        <td className='py-2 px-3'>{cat.description || '-'}</td>
                        <td className='py-2 px-3'>{cat.sort_order}</td>
                        <td className='py-2 px-3'>
                          <Tag color={cat.status === 1 ? 'green' : 'red'} shape='circle'>
                            {cat.status === 1 ? t('启用') : t('禁用')}
                          </Tag>
                        </td>
                        <td className='py-2 px-3 text-right'>
                          <Space>
                            <Button type='tertiary' size='small' onClick={() => {
                              setEditingCategory(cat);
                              setShowCatEdit(true);
                            }}>
                              {t('编辑')}
                            </Button>
                            <Popconfirm title={t('确定删除此分类吗？')} content={t('此操作不可撤销')} onConfirm={() => handleDeleteCategory(cat.id)}>
                              <Button type='danger' theme='light' size='small'>
                                {t('删除')}
                              </Button>
                            </Popconfirm>
                          </Space>
                        </td>
                      </tr>
                    ))}
                    {categories.length === 0 && (
                      <tr>
                        <td colSpan={6} className='py-8 text-center text-gray-400'>
                          {t('暂无数据')}
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </Spin>
          </Card>
        </Tabs.TabPane>
      </Tabs>
    </div>
  );
};

export default ArticleManagement;
