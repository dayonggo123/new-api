import React, { useState, useEffect, useCallback } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Table,
  Tag,
  Space,
  Modal,
  Pagination,
  Select,
  Typography,
  Empty,
  Progress,
} from '@douyinfe/semi-ui';
import {
  IconPlus,
  IconRefresh,
  IconLanguage,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';

const { Title, Text } = Typography;

const statusMap = {
  1: { color: 'green', text: '启用' },
  2: { color: 'red', text: '禁用' },
};

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
    obj[lang.code] = { name: '', system_prompt: '', user_prompt: '', description: '' };
  });
  return obj;
};

export default function PresetPrompt() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [modalVisible, setModalVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [formApi, setFormApi] = useState(null);
  const [editingItem, setEditingItem] = useState(null);
  const [categories, setCategories] = useState([]);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searchCategory, setSearchCategory] = useState('');
  const [searchStatus, setSearchStatus] = useState('');
  const [activeLang, setActiveLang] = useState(DEFAULT_LANG);
  const [i18nData, setI18nData] = useState(emptyI18n());
  const [translating, setTranslating] = useState(false);
  const [translateProgress, setTranslateProgress] = useState(0);

  const loadCategories = useCallback(async () => {
    try {
      const res = await API.get('/api/preset-prompt/categories/all');
      const { success, data: cats } = res.data;
      if (success) {
        setCategories(cats || []);
      }
    } catch (e) {
      // ignore
    }
  }, []);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const params = {
        p: page,
        page_size: pageSize,
      };
      if (searchKeyword) params.keyword = searchKeyword;
      if (searchCategory) params.category = searchCategory;
      if (searchStatus) params.status = searchStatus;

      const res = await API.get('/api/preset-prompt', { params });
      const { success, message, data: result } = res.data;
      if (success) {
        setData(result.items || []);
        setTotal(result.total || 0);
      } else {
        showError(message);
      }
    } catch (err) {
      showError(err.message);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, searchKeyword, searchCategory, searchStatus]);

  useEffect(() => {
    loadData();
    loadCategories();
  }, [loadData, loadCategories]);

  const handleAdd = () => {
    setEditingItem(null);
    setActiveLang(DEFAULT_LANG);
    setI18nData(emptyI18n());
    setModalVisible(true);
    if (formApi) {
      formApi.reset();
    }
  };

  const handleEdit = (record) => {
    setEditingItem(record);
    setActiveLang(DEFAULT_LANG);
    let parsed = {};
    try {
      if (record.i18n) {
        parsed = JSON.parse(record.i18n);
      }
    } catch (e) {
      parsed = {};
    }
    const merged = emptyI18n();
    Object.keys(parsed).forEach((code) => {
      if (merged[code]) {
        merged[code] = { ...merged[code], ...parsed[code] };
      }
    });
    setI18nData(merged);
    setModalVisible(true);
    if (formApi) {
      formApi.setValues({
        name: record.name,
        system_prompt: record.system_prompt,
        user_prompt: record.user_prompt,
        description: record.description,
        category: record.category,
        status: record.status,
        sort_order: record.sort_order,
      });
    }
  };

  const handleDelete = (record) => {
    Modal.confirm({
      title: t('确认删除'),
      content: `${t('确定要删除预设提示词')}「${record.name}」？`,
      onOk: async () => {
        try {
          const res = await API.delete(`/api/preset-prompt/${record.id}`);
          if (res.data.success) {
            showSuccess(t('删除成功'));
            loadData();
          } else {
            showError(res.data.message);
          }
        } catch (err) {
          showError(err.message);
        }
      },
    });
  };

  const handleAutoTranslate = async () => {
    if (!formApi) return;
    const values = formApi.getValues();
    const items = [
      { key: 'name', text: values.name || '' },
      { key: 'system_prompt', text: values.system_prompt || '' },
      { key: 'user_prompt', text: values.user_prompt || '' },
      { key: 'description', text: values.description || '' },
    ].filter((item) => item.text !== '');
    if (items.length === 0) {
      showError(t('请至少填写一个默认语言字段后再翻译'));
      return;
    }
    const targetLangs = LANGUAGES.filter((l) => l.code !== DEFAULT_LANG).map((l) => l.code);
    setTranslating(true);
    setTranslateProgress(0);
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
            updated[langCode] = { ...updated[langCode], ...result[langCode] };
          }
        });
        setI18nData(updated);
        showSuccess(t('翻译完成'));
      } else {
        showError(message || t('翻译失败'));
      }
    } catch (err) {
      showError(err.message);
    } finally {
      setTranslating(false);
      setTranslateProgress(100);
    }
  };

  const handleSubmit = async () => {
    if (!formApi) return;
    const values = await formApi.validate();
    setSubmitting(true);
    try {
      const payload = {
        name: values.name,
        system_prompt: values.system_prompt || '',
        user_prompt: values.user_prompt || '',
        description: values.description || '',
        category: values.category || '',
        status: values.status || 1,
        sort_order: values.sort_order || 0,
        i18n: JSON.stringify(i18nData),
      };

      let res;
      if (editingItem) {
        res = await API.put('/api/preset-prompt', { ...payload, id: editingItem.id });
      } else {
        res = await API.post('/api/preset-prompt', payload);
      }

      if (res.data.success) {
        showSuccess(editingItem ? t('更新成功') : t('创建成功'));
        setModalVisible(false);
        loadData();
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  const updateI18nField = (langCode, field, value) => {
    setI18nData((prev) => ({
      ...prev,
      [langCode]: {
        ...prev[langCode],
        [field]: value,
      },
    }));
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: t('名称'),
      dataIndex: 'name',
      width: 200,
      render: (text) => <Text strong>{text}</Text>,
    },
    {
      title: t('分类'),
      dataIndex: 'category',
      width: 120,
      render: (text) => text || '-',
    },
    {
      title: t('系统提示词'),
      dataIndex: 'system_prompt',
      ellipsis: true,
      width: 200,
      render: (text) => text ? text.slice(0, 30) + (text.length > 30 ? '...' : '') : '-',
    },
    {
      title: t('用户提示词'),
      dataIndex: 'user_prompt',
      ellipsis: true,
      width: 200,
      render: (text) => text ? text.slice(0, 30) + (text.length > 30 ? '...' : '') : '-',
    },
    {
      title: t('描述'),
      dataIndex: 'description',
      ellipsis: true,
      render: (text) => text || '-',
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 100,
      render: (status) => {
        const s = statusMap[status];
        return <Tag color={s?.color}>{t(s?.text)}</Tag>;
      },
    },
    {
      title: t('排序'),
      dataIndex: 'sort_order',
      width: 80,
    },
    {
      title: t('操作'),
      width: 160,
      render: (_, record) => (
        <Space>
          <Button type='primary' size='small' onClick={() => handleEdit(record)}>
            {t('编辑')}
          </Button>
          <Button type='danger' size='small' onClick={() => handleDelete(record)}>
            {t('删除')}
          </Button>
        </Space>
      ),
    },
  ];

  const renderLangFields = (langCode) => {
    const isDefault = langCode === DEFAULT_LANG;
    if (isDefault) {
      return (
        <>
          <Form.Input
            field='name'
            label={t('名称')}
            placeholder={t('请输入预设提示词名称')}
            rules={[{ required: true, message: t('名称不能为空') }]}
          />
          <Form.TextArea
            field='system_prompt'
            label={t('系统提示词')}
            placeholder={t('请输入系统提示词（可选）')}
            rows={4}
          />
          <Form.TextArea
            field='user_prompt'
            label={t('用户提示词')}
            placeholder={t('请输入用户提示词（可选）')}
            rows={4}
          />
          <Form.TextArea
            field='description'
            label={t('描述')}
            placeholder={t('请输入描述（可选）')}
            rows={2}
          />
        </>
      );
    }
    const data = i18nData[langCode] || {};
    const langLabel = LANGUAGES.find((l) => l.code === langCode)?.label || langCode;
    return (
      <>
        <div style={{ marginBottom: 12 }}>
          <label style={{ display: 'block', marginBottom: 4, fontWeight: 500, color: 'var(--semi-color-text-0)' }}>
            {t('名称')}
          </label>
          <Input
            value={data.name || ''}
            onChange={(v) => updateI18nField(langCode, 'name', v)}
            placeholder={t('请输入') + ' ' + langLabel}
          />
        </div>
        <div style={{ marginBottom: 12 }}>
          <label style={{ display: 'block', marginBottom: 4, fontWeight: 500, color: 'var(--semi-color-text-0)' }}>
            {t('系统提示词')}
          </label>
          <Input.TextArea
            value={data.system_prompt || ''}
            onChange={(v) => updateI18nField(langCode, 'system_prompt', v)}
            placeholder={t('请输入系统提示词（可选）')}
            rows={4}
          />
        </div>
        <div style={{ marginBottom: 12 }}>
          <label style={{ display: 'block', marginBottom: 4, fontWeight: 500, color: 'var(--semi-color-text-0)' }}>
            {t('用户提示词')}
          </label>
          <Input.TextArea
            value={data.user_prompt || ''}
            onChange={(v) => updateI18nField(langCode, 'user_prompt', v)}
            placeholder={t('请输入用户提示词（可选）')}
            rows={4}
          />
        </div>
        <div style={{ marginBottom: 12 }}>
          <label style={{ display: 'block', marginBottom: 4, fontWeight: 500, color: 'var(--semi-color-text-0)' }}>
            {t('描述')}
          </label>
          <Input.TextArea
            value={data.description || ''}
            onChange={(v) => updateI18nField(langCode, 'description', v)}
            placeholder={t('请输入描述（可选）')}
            rows={2}
          />
        </div>
      </>
    );
  };

  return (
    <div className='mt-[60px] px-4 py-6'>
      <Card
        title={
          <div className='flex items-center justify-between'>
            <Title heading={4}>{t('预设提示词管理')}</Title>
            <Space>
              <Button icon={<IconRefresh />} onClick={loadData}>
                {t('刷新')}
              </Button>
              <Button type='primary' icon={<IconPlus />} onClick={handleAdd}>
                {t('新增')}
              </Button>
            </Space>
          </div>
        }
      >
        <div className='mb-4 flex flex-wrap gap-3'>
          <Input
            placeholder={t('搜索名称/描述')}
            value={searchKeyword}
            onChange={(v) => setSearchKeyword(v)}
            onPressEnter={() => { setPage(1); loadData(); }}
            style={{ width: 220 }}
          />
          <Select
            placeholder={t('全部分类')}
            value={searchCategory}
            onChange={(v) => { setSearchCategory(v); setPage(1); }}
            style={{ width: 160 }}
            allowClear
            optionList={categories.map((cat) => ({ value: cat, label: cat }))}
          />
          <Select
            placeholder={t('全部状态')}
            value={searchStatus}
            onChange={(v) => { setSearchStatus(v); setPage(1); }}
            style={{ width: 140 }}
            allowClear
            optionList={[
              { value: 1, label: t('启用') },
              { value: 2, label: t('禁用') },
            ]}
          />
          <Button type='primary' onClick={() => { setPage(1); loadData(); }}>
            {t('查询')}
          </Button>
        </div>

        <Table
          columns={columns}
          dataSource={data}
          loading={loading}
          pagination={false}
          rowKey='id'
          empty={<Empty description={t('暂无数据')} />}
        />

        <div className='mt-4 flex justify-end'>
          <Pagination
            total={total}
            currentPage={page}
            pageSize={pageSize}
            onPageChange={(p) => setPage(p)}
            onPageSizeChange={(size) => { setPageSize(size); setPage(1); }}
            showSizeChanger
          />
        </div>
      </Card>

      <Modal
        title={
          <div className='flex items-center justify-between' style={{ paddingRight: 32 }}>
            <span>{editingItem ? t('编辑预设提示词') : t('新增预设提示词')}</span>
            <Button
              icon={<IconLanguage />}
              loading={translating}
              onClick={handleAutoTranslate}
              type='tertiary'
              size='small'
            >
              {t('自动翻译')}
            </Button>
          </div>
        }
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        confirmLoading={submitting}
        centered
        maskClosable={false}
        style={{ width: 700 }}
        footer={
          <div className='flex items-center justify-between'>
            <div style={{ flex: 1 }}>
              {translating && (
                <Progress
                  percent={translateProgress}
                  size='small'
                  showInfo
                  style={{ width: 200 }}
                />
              )}
            </div>
            <Space>
              <Button onClick={() => setModalVisible(false)}>{t('取消')}</Button>
              <Button type='primary' loading={submitting} onClick={handleSubmit}>
                {t('确定')}
              </Button>
            </Space>
          </div>
        }
      >
        <Form
          getFormApi={(api) => setFormApi(api)}
          initValues={{
            status: 1,
            sort_order: 0,
          }}
        >
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
              const isActive = activeLang === lang.code;
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
                    borderBottom: isActive ? '2px solid var(--semi-color-primary)' : '2px solid transparent',
                    color: isActive ? 'var(--semi-color-primary)' : 'var(--semi-color-text-2)',
                    fontWeight: isActive ? 600 : 400,
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

          {/* 语言字段 */}
          <div style={{ paddingTop: 4, paddingBottom: 8 }}>
            {renderLangFields(activeLang)}
          </div>

          <Form.Input
            field='category'
            label={t('分类')}
            placeholder={t('请输入分类（可选）')}
          />
          <Form.Select
            field='status'
            label={t('状态')}
            rules={[{ required: true }]}
            optionList={[
              { value: 1, label: t('启用') },
              { value: 2, label: t('禁用') },
            ]}
          />
          <Form.InputNumber
            field='sort_order'
            label={t('排序')}
            placeholder={t('数字越大越靠前')}
            min={0}
          />
        </Form>
      </Modal>
    </div>
  );
}
