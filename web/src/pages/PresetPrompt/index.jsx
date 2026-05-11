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
  Tabs,
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
const { Option } = Select;
const { TabPane } = Tabs;

const statusMap = {
  1: { color: 'green', text: '启用' },
  2: { color: 'red', text: '禁用' },
};

// 12 种支持语言（第一项为默认语言中文）
const LANGUAGES = [
  { code: 'zh-CN', label: '中文（默认）' },
  { code: 'en', label: 'English' },
  { code: 'fr', label: 'Français' },
  { code: 'ru', label: 'Русский' },
  { code: 'ja', label: '日本語' },
  { code: 'vi', label: 'Tiếng Việt' },
  { code: 'zh-TW', label: '繁體中文' },
  { code: 'es', label: 'Español' },
  { code: 'de', label: 'Deutsch' },
  { code: 'ko', label: '한국어' },
  { code: 'pt', label: 'Português' },
  { code: 'it', label: 'Italiano' },
];

const DEFAULT_LANG = 'zh-CN';

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
  const [i18nData, setI18nData] = useState({});

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
    setI18nData({});
    setActiveLang(DEFAULT_LANG);
    setModalVisible(true);
    if (formApi) formApi.reset();
  };

  const handleEdit = (record) => {
    setEditingItem(record);
    setActiveLang(DEFAULT_LANG);
    // 解析 i18n JSON
    let parsed = {};
    if (record.i18n) {
      try {
        parsed = JSON.parse(record.i18n);
      } catch (e) {
        parsed = {};
      }
    }
    setI18nData(parsed);
    setModalVisible(true);
  };

  // 当 Modal 打开且 formApi 就绪时填充表单值
  useEffect(() => {
    if (modalVisible && editingItem && formApi) {
      formApi.setValues({
        name: editingItem.name,
        system_prompt: editingItem.system_prompt,
        user_prompt: editingItem.user_prompt,
        description: editingItem.description,
        category: editingItem.category,
        status: editingItem.status,
        sort_order: editingItem.sort_order,
      });
    }
  }, [modalVisible, editingItem, formApi]);

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
    const items = [];
    if (values.name?.trim()) items.push({ key: 'name', text: values.name.trim() });
    if (values.system_prompt?.trim()) items.push({ key: 'system_prompt', text: values.system_prompt.trim() });
    if (values.user_prompt?.trim()) items.push({ key: 'user_prompt', text: values.user_prompt.trim() });
    if (values.description?.trim()) items.push({ key: 'description', text: values.description.trim() });

    if (items.length === 0) {
      showError('请先填写中文内容');
      return;
    }

    try {
      const res = await API.post('/api/translate/batch', {
        items,
        source_lang: 'ZH',
        target_langs: ['EN', 'FR', 'RU', 'JA', 'VI', 'ZH-TW', 'ES', 'DE', 'KO', 'PT', 'IT'],
      });
      if (res.data.success) {
        // 后端返回的语言 key 是大写的（EN/FR/…），但 i18nData 和 Form 字段都用小写
        const normalized = {};
        Object.entries(res.data.data).forEach(([lang, fields]) => {
          normalized[lang.toLowerCase()] = fields;
        });
        setI18nData(normalized);
        // 同步翻译结果到表单字段（字段已挂载，initValue 不会自动更新）
        Object.entries(normalized).forEach(([lang, fields]) => {
          Object.entries(fields).forEach(([key, value]) => {
            formApi.setValue(`${key}_${lang}`, value);
          });
        });
        showSuccess('翻译完成，已填充到各语言 Tab');
      } else {
        showError(res.data.message || '翻译失败');
      }
    } catch (err) {
      showError(err.message || '翻译服务不可用');
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

  // 获取当前语言表单值
  const getLangField = (field) => {
    if (activeLang === DEFAULT_LANG) {
      return formApi?.getValue(field) || '';
    }
    return i18nData[activeLang]?.[field] || '';
  };

  // 设置当前语言表单值
  const setLangField = (field, value) => {
    if (activeLang === DEFAULT_LANG) {
      formApi?.setValue(field, value);
    } else {
      setI18nData((prev) => ({
        ...prev,
        [activeLang]: {
          ...prev[activeLang],
          [field]: value,
        },
      }));
    }
  };

  // 判断某语言是否有翻译
  const hasTranslation = (code) => {
    if (code === DEFAULT_LANG) return true;
    const t = i18nData[code];
    return t && (t.name || t.system_prompt || t.user_prompt || t.description);
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
          >
            {categories.map((cat) => (
              <Option key={cat} value={cat}>{cat}</Option>
            ))}
          </Select>
          <Select
            placeholder={t('全部状态')}
            value={searchStatus}
            onChange={(v) => { setSearchStatus(v); setPage(1); }}
            style={{ width: 140 }}
            allowClear
          >
            <Option value={1}>{t('启用')}</Option>
            <Option value={2}>{t('禁用')}</Option>
          </Select>
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
        title={editingItem ? t('编辑预设提示词') : t('新增预设提示词')}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        confirmLoading={submitting}
        centered
        maskClosable={false}
        width={800}
        bodyStyle={{ maxHeight: '75vh', overflow: 'auto', paddingRight: 4 }}
      >
        <Tabs
          activeKey={activeLang}
          onChange={(key) => setActiveLang(key)}
          type='button'
          style={{ marginBottom: 8 }}
        >
          {LANGUAGES.map((lang) => (
            <TabPane
              tab={
                <span>
                  {lang.label}
                  {hasTranslation(lang.code) && lang.code !== DEFAULT_LANG && (
                    <span style={{ color: 'var(--semi-color-primary)', marginLeft: 4, fontSize: 8 }}>●</span>
                  )}
                </span>
              }
              itemKey={lang.code}
              key={lang.code}
            />
          ))}
        </Tabs>

        <div style={{ marginBottom: 16 }}>
          <Button
            type='tertiary'
            icon={<IconLanguage />}
            onClick={handleAutoTranslate}
          >
            {t('自动翻译')}（DeepLX）
          </Button>
          <Text type='tertiary' size='small' style={{ marginLeft: 8 }}>
            {t('将中文内容一键翻译成其他 11 种语言')}
          </Text>
        </div>

        <Form
          getFormApi={(api) => setFormApi(api)}
          initValues={{
            status: 1,
            sort_order: 0,
          }}
        >
          <div style={{ display: activeLang === DEFAULT_LANG ? 'block' : 'none' }}>
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
          </div>

          {LANGUAGES.filter(l => l.code !== DEFAULT_LANG).map((lang) => (
            <div key={lang.code} style={{ display: activeLang === lang.code ? 'block' : 'none' }}>
              <Form.Section text={`${lang.label} ${t('翻译')}`}>
                <Form.Input
                  field={`name_${lang.code}`}
                  label={t('名称')}
                  placeholder={t('请输入翻译后的名称')}
                  initValue={i18nData[lang.code]?.name || ''}
                  onChange={(v) => setLangField('name', v)}
                />
                <Form.TextArea
                  field={`system_prompt_${lang.code}`}
                  label={t('系统提示词')}
                  placeholder={t('请输入翻译后的系统提示词')}
                  rows={4}
                  initValue={i18nData[lang.code]?.system_prompt || ''}
                  onChange={(v) => setLangField('system_prompt', v)}
                />
                <Form.TextArea
                  field={`user_prompt_${lang.code}`}
                  label={t('用户提示词')}
                  placeholder={t('请输入翻译后的用户提示词')}
                  rows={4}
                  initValue={i18nData[lang.code]?.user_prompt || ''}
                  onChange={(v) => setLangField('user_prompt', v)}
                />
                <Form.TextArea
                  field={`description_${lang.code}`}
                  label={t('描述')}
                  placeholder={t('请输入翻译后的描述')}
                  rows={2}
                  initValue={i18nData[lang.code]?.description || ''}
                  onChange={(v) => setLangField('description', v)}
                />
              </Form.Section>
            </div>
          ))}

          <Form.Input
            field='category'
            label={t('分类')}
            placeholder={t('请输入分类（可选）')}
          />
          <Form.Select
            field='status'
            label={t('状态')}
            rules={[{ required: true }]}
          >
            <Option value={1}>{t('启用')}</Option>
            <Option value={2}>{t('禁用')}</Option>
          </Form.Select>
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
