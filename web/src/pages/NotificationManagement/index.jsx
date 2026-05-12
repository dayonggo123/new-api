import React, { useState, useEffect, useCallback, useRef } from 'react';
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
  Tooltip,
  Banner,
  Checkbox,
  Tabs,
  TextArea,
} from '@douyinfe/semi-ui';
import {
  IconPlus,
  IconRefresh,
  IconHelpCircle,
  IconLanguage,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';

const { Title, Text } = Typography;

const typeMap = {
  system: { color: 'blue', text: '系统' },
  promotion: { color: 'green', text: '活动' },
  announcement: { color: 'orange', text: '公告' },
  task_status: { color: 'purple', text: '任务' },
};

const targetTypeMap = {
  all: '全员广播',
  users: '指定用户',
  group: '指定分组',
  tier: '按层级',
  tag: '按标签',
};

const templateVars = [
  { key: '{{username}}', desc: '用户名' },
  { key: '{{display_name}}', desc: '显示名称' },
  { key: '{{tier_name}}', desc: '层级名称' },
  { key: '{{tier_level}}', desc: '层级编号' },
  { key: '{{total_points}}', desc: '总积分' },
  { key: '{{consecutive_days}}', desc: '连续签到天数' },
];

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

export default function NotificationManagement() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [modalVisible, setModalVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [formApi, setFormApi] = useState(null);
  const contentRef = useRef(null);
  const [targetType, setTargetType] = useState('all');
  const [useTemplate, setUseTemplate] = useState(false);
  const [activeLang, setActiveLang] = useState(DEFAULT_LANG);
  const [i18nData, setI18nData] = useState({});
  const [editingItem, setEditingItem] = useState(null);

  // tier / tag options
  const [tiers, setTiers] = useState([]);
  const [tags, setTags] = useState([]);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/admin/notifications', {
        params: {
          page,
          page_size: pageSize,
        },
      });
      const { success, data: result } = res.data;
      if (success) {
        setData(result.items || []);
        setTotal(result.total || 0);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
    setLoading(false);
  }, [page, pageSize]);

  const loadTiers = useCallback(async () => {
    try {
      const res = await API.get('/api/admin/tiers');
      if (res.data.success) {
        setTiers(res.data.data || []);
      }
    } catch (e) {
      // ignore
    }
  }, []);

  const loadTags = useCallback(async () => {
    try {
      const res = await API.get('/api/admin/tags');
      if (res.data.success) {
        setTags(res.data.data || []);
      }
    } catch (e) {
      // ignore
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  useEffect(() => {
    loadTiers();
    loadTags();
  }, [loadTiers, loadTags]);

  const handleOpenModal = () => {
    setEditingItem(null);
    setI18nData({});
    setActiveLang(DEFAULT_LANG);
    setTargetType('all');
    setUseTemplate(false);
    setModalVisible(true);
  };

  const handleCloseModal = () => {
    setModalVisible(false);
    setEditingItem(null);
    setTargetType('all');
    setUseTemplate(false);
    setI18nData({});
    setActiveLang(DEFAULT_LANG);
  };

  const handleEdit = (record) => {
    setEditingItem(record);
    setActiveLang(DEFAULT_LANG);
    setUseTemplate(false);

    let parsed = {};
    try {
      if (record.i18n) {
        parsed = JSON.parse(record.i18n);
      }
    } catch (e) {
      parsed = {};
    }
    setI18nData(parsed);
    setModalVisible(true);
  };

  const handleAutoTranslate = async () => {
    if (!formApi) return;
    const values = formApi.getValues();
    const items = [];
    if (values.title?.trim()) items.push({ key: 'title', text: values.title.trim() });
    if (values.content?.trim()) items.push({ key: 'content', text: values.content.trim() });

    if (items.length === 0) {
      showError('请先填写中文标题和内容');
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
        // 同步翻译结果到表单字段
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

  const handleRetranslate = async (targetLang) => {
    if (!formApi) return;
    const values = formApi.getValues();
    const items = [];
    if (values.title?.trim()) items.push({ key: 'title', text: values.title.trim() });
    if (values.content?.trim()) items.push({ key: 'content', text: values.content.trim() });

    if (items.length === 0) {
      showError('请先填写中文标题和内容');
      return;
    }

    try {
      const res = await API.post('/api/translate/batch', {
        items,
        source_lang: 'ZH',
        target_langs: [targetLang.toUpperCase()],
      });
      if (res.data.success) {
        const normalized = {};
        Object.entries(res.data.data).forEach(([lang, fields]) => {
          normalized[lang.toLowerCase()] = fields;
        });
        setI18nData((prev) => ({ ...prev, ...normalized }));
        Object.entries(normalized).forEach(([lang, fields]) => {
          Object.entries(fields).forEach(([key, value]) => {
            formApi.setValue(`${key}_${lang}`, value);
          });
        });
        showSuccess('翻译完成');
      } else {
        showError(res.data.message || '翻译失败');
      }
    } catch (err) {
      showError(err.message || '翻译服务不可用');
    }
  };

  // Modal 打开且 Form 挂载完成后设置值
  useEffect(() => {
    if (!modalVisible || !formApi) return;
    if (editingItem) {
      formApi.setValues({
        title: editingItem.title,
        content: editingItem.content,
        type: editingItem.type,
        action_url: editingItem.action_url || '',
      });
      // 同步多语言字段到表单
      Object.entries(i18nData).forEach(([lang, fields]) => {
        if (fields) {
          Object.entries(fields).forEach(([key, value]) => {
            if (value) {
              formApi.setValue(`${key}_${lang}`, value);
            }
          });
        }
      });
    } else {
      formApi.reset();
    }
  }, [modalVisible, formApi, editingItem]);

  const handleSubmit = async (values) => {
    setSubmitting(true);
    try {
      const payload = {
        title: values.title,
        content: values.content,
        i18n: JSON.stringify(i18nData),
        type: values.type,
        action_url: values.action_url || '',
      };

      let res;
      if (editingItem) {
        res = await API.put(`/api/admin/notifications/${editingItem.id}`, payload);
      } else {
        const createPayload = {
          ...payload,
          use_template: !!useTemplate,
          target_type: values.target_type,
        };
        // handle array values from multi-select / tag-input, convert to int arrays
        if (values.target_users) {
          createPayload.target_users = Array.isArray(values.target_users)
            ? values.target_users.map(Number).filter((v) => !isNaN(v))
            : [Number(values.target_users)].filter((v) => !isNaN(v));
        }
        if (values.target_tiers) {
          createPayload.target_tiers = Array.isArray(values.target_tiers)
            ? values.target_tiers.map(Number).filter((v) => !isNaN(v))
            : [Number(values.target_tiers)].filter((v) => !isNaN(v));
        }
        if (values.target_tags) {
          createPayload.target_tags = Array.isArray(values.target_tags)
            ? values.target_tags.map(Number).filter((v) => !isNaN(v))
            : [Number(values.target_tags)].filter((v) => !isNaN(v));
        }
        res = await API.post('/api/admin/notifications', createPayload);
      }

      const { success, message } = res.data;
      if (success) {
        showSuccess(message || (editingItem ? '更新成功' : '发送成功'));
        handleCloseModal();
        loadData();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
    setSubmitting(false);
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
    const tr = i18nData[code];
    return tr && (tr.title || tr.content);
  };

  // 插入模板变量到内容框
  const insertTemplateVar = (varKey) => {
    const textarea = contentRef.current;
    const fieldName = activeLang === DEFAULT_LANG ? 'content' : `content_${activeLang}`;
    const current = activeLang === DEFAULT_LANG
      ? (formApi?.getValue('content') || '')
      : (i18nData[activeLang]?.content || '');
    let start = current.length;
    let end = current.length;
    if (textarea) {
      start = textarea.selectionStart ?? start;
      end = textarea.selectionEnd ?? end;
    }
    const newValue = current.slice(0, start) + varKey + current.slice(end);
    setLangField('content', newValue);
    setTimeout(() => {
      if (textarea) {
        textarea.focus();
        const pos = start + varKey.length;
        textarea.setSelectionRange(pos, pos);
      }
    }, 0);
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: '标题',
      dataIndex: 'title',
      ellipsis: true,
    },
    {
      title: '内容',
      dataIndex: 'content',
      ellipsis: true,
      render: (text) => (
        <span style={{ maxWidth: 300, display: 'inline-block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {text}
        </span>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 100,
      render: (type) => {
        const config = typeMap[type] || { color: 'grey', text: type };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '目标',
      dataIndex: 'user_id',
      width: 120,
      render: (userId) => (
        <Tag color={userId === 0 ? 'red' : 'blue'}>
          {userId === 0 ? '全员广播' : `用户 ${userId}`}
        </Tag>
      ),
    },
    {
      title: '已读',
      dataIndex: 'is_read',
      width: 80,
      render: (isRead) => (
        <Tag color={isRead ? 'green' : 'orange'}>
          {isRead ? '已读' : '未读'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_time',
      width: 160,
      render: (time) => {
        if (!time) return '-';
        const date = new Date(time * 1000);
        return date.toLocaleString('zh-CN');
      },
    },
    {
      title: '操作',
      width: 100,
      render: (_, record) => (
        <Button type='primary' size='small' onClick={() => handleEdit(record)}>
          编辑
        </Button>
      ),
    },
  ];

  const renderTargetFields = () => {
    switch (targetType) {
      case 'users':
        return (
          <Form.TagInput
            field='target_users'
            label='目标用户 ID'
            rules={[{ required: true, message: '请至少输入一个用户 ID' }]}
            placeholder='输入用户 ID 后按回车'
          />
        );
      case 'group':
        return (
          <Form.Input
            field='target_group'
            label='目标用户组'
            rules={[{ required: true, message: '请输入用户组名' }]}
            placeholder='如：default'
          />
        );
      case 'tier':
        return (
          <Form.Select
            field='target_tiers'
            label='目标层级'
            rules={[{ required: true, message: '请选择至少一个层级' }]}
            placeholder='选择目标层级'
            multiple
            filter
            optionList={tiers.map((tier) => ({
              value: tier.level,
              label: <Tag color={tier.color || 'blue'}>L{tier.level} {tier.name}</Tag>,
            }))}
          />
        );
      case 'tag':
        return (
          <Form.Select
            field='target_tags'
            label='目标标签'
            rules={[{ required: true, message: '请选择至少一个标签' }]}
            placeholder='选择目标标签'
            multiple
            filter
            optionList={tags.map((tag) => ({
              value: tag.id,
              label: <Tag color={tag.color || 'blue'}>{tag.name}</Tag>,
            }))}
          />
        );
      default:
        return null;
    }
  };

  return (
    <div style={{ padding: '84px 20px 20px' }}>
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <Title heading={4} style={{ margin: 0 }}>消息管理</Title>
          </div>
        }
        headerExtraContent={
          <Space>
            <Button
              icon={<IconRefresh />}
              onClick={loadData}
              loading={loading}
            >
              刷新
            </Button>
            <Button
              type='primary'
              icon={<IconPlus />}
              onClick={handleOpenModal}
            >
              发布消息
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={data}
          loading={loading}
          pagination={false}
          empty={<Empty description='暂无消息' />}
          rowKey='id'
        />
        <div style={{ marginTop: 16, display: 'flex', justifyContent: 'flex-end' }}>
          <Pagination
            total={total}
            currentPage={page}
            pageSize={pageSize}
            onPageChange={(p) => setPage(p)}
            onPageSizeChange={(s) => { setPageSize(s); setPage(1); }}
          />
        </div>
      </Card>

      <Modal
        title={editingItem ? '编辑消息' : '发布消息'}
        visible={modalVisible}
        onCancel={handleCloseModal}
        footer={null}
        width={800}
        bodyStyle={{ maxHeight: '75vh', overflow: 'auto', paddingRight: 4 }}
      >
        <Tabs
          activeKey={activeLang}
          onChange={(key) => setActiveLang(key)}
          type='button'
          style={{ marginBottom: 8 }}
          tabList={LANGUAGES.map((lang) => ({
            tab: (
              <span>
                {lang.label}
                {hasTranslation(lang.code) && lang.code !== DEFAULT_LANG && (
                  <span style={{ color: 'var(--semi-color-primary)', marginLeft: 4, fontSize: 8 }}>●</span>
                )}
              </span>
            ),
            itemKey: lang.code,
          }))}
        />

        <div style={{ marginBottom: 16 }}>
          <Button
            type='tertiary'
            icon={<IconLanguage />}
            onClick={handleAutoTranslate}
          >
            自动翻译（AI）
          </Button>
          <Text type='tertiary' size='small' style={{ marginLeft: 8 }}>
            将中文内容一键翻译成其他 11 种语言
          </Text>
        </div>

        <Form
          getFormApi={(api) => setFormApi(api)}
          onSubmit={handleSubmit}
          layout='vertical'
        >
          <div style={{ display: activeLang === DEFAULT_LANG ? 'block' : 'none' }}>
            <Form.Input
              field='title'
              label='标题'
              rules={[{ required: true, message: '请输入标题' }]}
              placeholder='请输入消息标题'
            />
            <Form.TextArea
              field='content'
              label='内容'
              rules={[{ required: true, message: '请输入内容' }]}
              placeholder='请输入消息内容'
              rows={4}
              ref={contentRef}
            />
          </div>

          {LANGUAGES.filter(l => l.code !== DEFAULT_LANG).map((lang) => (
            <div key={lang.code} style={{ display: activeLang === lang.code ? 'block' : 'none' }}>
              <div style={{ marginBottom: 12 }}>
                <Button
                  icon={<IconLanguage />}
                  type='tertiary'
                  size='small'
                  onClick={() => handleRetranslate(lang.code)}
                  loading={translating}
                >
                  重新翻译
                </Button>
              </div>
              <Form.Input
                field={`title_${lang.code}`}
                label={`${lang.label} 标题`}
                placeholder='请输入翻译后的标题'
                initValue={i18nData[lang.code]?.title || ''}
                onChange={(v) => setLangField('title', v)}
              />
              <Form.TextArea
                field={`content_${lang.code}`}
                label={`${lang.label} 内容`}
                placeholder='请输入翻译后的内容'
                rows={4}
                initValue={i18nData[lang.code]?.content || ''}
                onChange={(v) => setLangField('content', v)}
                ref={contentRef}
              />
            </div>
          ))}

          {!editingItem && (
            <>
              {useTemplate && (
                <Banner
                  type='info'
                  icon={<IconHelpCircle />}
                  title='模板变量可用（点击插入）'
                  description={
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, marginTop: 8 }}>
                      {templateVars.map((v) => (
                        <Tooltip content={v.desc} key={v.key}>
                          <Tag
                            color='blue'
                            style={{ cursor: 'pointer' }}
                            onClick={() => insertTemplateVar(v.key)}
                          >
                            {v.key}
                          </Tag>
                        </Tooltip>
                      ))}
                    </div>
                  }
                  closeIcon={null}
                />
              )}

              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 12 }}>
                <Checkbox
                  checked={useTemplate}
                  onChange={(e) => setUseTemplate(e.target.checked)}
                >
                  启用模板变量替换
                </Checkbox>
                <Tooltip
                  content={
                    <div>
                      <div>开启后，消息内容中的变量会自动替换为每个用户的实际值：</div>
                      {templateVars.map((v) => (
                        <div key={v.key}>{v.key} = {v.desc}</div>
                      ))}
                    </div>
                  }
                >
                  <IconHelpCircle style={{ color: 'var(--semi-color-text-2)', cursor: 'pointer' }} />
                </Tooltip>
              </div>
            </>
          )}

          <Form.Select
            field='type'
            label='类型'
            rules={[{ required: true, message: '请选择类型' }]}
            placeholder='请选择消息类型'
            initValue='system'
            optionList={[
              { value: 'system', label: '系统' },
              { value: 'promotion', label: '活动' },
              { value: 'announcement', label: '公告' },
              { value: 'task_status', label: '任务' },
            ]}
          />

          {!editingItem && (
            <>
              <Form.Select
                field='target_type'
                label='发送目标'
                rules={[{ required: true, message: '请选择发送目标' }]}
                placeholder='请选择发送目标'
                initValue='all'
                onChange={(value) => setTargetType(value)}
                optionList={[
                  { value: 'all', label: '全员广播' },
                  { value: 'users', label: '指定用户' },
                  { value: 'group', label: '指定分组' },
                  { value: 'tier', label: '按层级' },
                  { value: 'tag', label: '按标签' },
                ]}
              />

              {renderTargetFields()}
            </>
          )}

          <Form.Input
            field='action_url'
            label='跳转链接'
            placeholder='可选：点击消息后的跳转链接'
          />
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 12, marginTop: 20 }}>
            <Button onClick={handleCloseModal}>取消</Button>
            <Button type='primary' htmlType='submit' loading={submitting}>
              {editingItem ? '更新' : '发布'}
            </Button>
          </div>
        </Form>
      </Modal>
    </div>
  );
}
