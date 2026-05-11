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

import React, { useEffect, useState, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  showSuccess,
  verifyJSONPromise,
} from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import {
  Button,
  SideSheet,
  Space,
  Spin,
  Typography,
  Card,
  Tag,
  Form,
  Avatar,
  Row,
  Col,
  Upload,
  Tabs,
  Input,
} from '@douyinfe/semi-ui';
import {
  IconSave,
  IconClose,
  IconBookStroked,
  IconUpload,
  IconLanguage,
} from '@douyinfe/semi-icons';

const { Text, Title } = Typography;


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

const EditPromptModal = (props) => {
  const { t } = useTranslation();
  const isEdit = props.editingPrompt.id !== undefined;
  const [loading, setLoading] = useState(isEdit);
  const isMobile = useIsMobile();
  const formApiRef = useRef(null);
  const [activeLang, setActiveLang] = useState(DEFAULT_LANG);
  const [i18nData, setI18nData] = useState({});
  const [translating, setTranslating] = useState(false);
  const [translateProgress, setTranslateProgress] = useState('');

  const PRESET_TAGS = [
    '电影感', '超写实', 'photography', 'nature', 'portrait', 'landscape',
    '写实', 'vehicle', 'character', 'minimalist', 'fashion', '自拍',
    '高级感', '时尚', '极简风', '人像', '辣妹', '胶片感', '街拍',
    'logo', '金发', '少女', 'interior', 'typography', 'paper-craft',
    'illustration', 'branding', '极简', '超现实', 'cartoon', 'product',
    '比基尼', '微距', '光影', '复古风', '复古', '3d', '奢华', 'food',
    'retro', 'poster', '氛围感', '特写', 'architecture', '高定', '质感',
    '闪光灯', '红发', '写真', '夜景', 'neon', '运动风', '治愈系', 'toy',
    '电影质感', '杂志风', '微缩', '性感', '唯美', 'creative',
    'futuristic', '时尚大片', '九宫格', '写实风', '美食', '信息图',
    '慵懒风', '情侣', '奇幻', '健身房', '暗黑风', 'animal', '少女感',
    '大片感', '肖像', '霓虹', '回眸', '黑白', '梦幻', '皮克斯', '柔光',
    '街头风', '美女', 'ui', '夏日', '奢华风', '抓拍', '飞溅', '居家',
    '霓虹灯', '红裙', '海报', '科幻', 'fantasy', '赛博风', '纯欲风',
    '四宫格', '千禧风', '雨夜', '浪漫',
  ];

  const getInitValues = () => ({
    title: '',
    content: '',
    content_en: '',
    description: '',
    cover_image_url: '',
    author: '',
    model: '',
    category_id: '',
    media_type: 'image',
    variables: '',
    tags: [],
    sort_order: 0,
    status: true,
  });

  const handleCancel = () => {
    props.handleClose();
  };

  const handleCoverUpload = async ({ fileInstance, onSuccess, onError }) => {
    const formData = new FormData();
    formData.append('images', fileInstance);
    try {
      const res = await API.post('/uapi/v1/upload_images', formData);
      if (res.data.urls && res.data.urls.length > 0) {
        onSuccess(res.data);
        formApiRef.current?.setValue('cover_image_url', res.data.urls[0]);
        showSuccess(t('封面上传成功'));
      } else {
        onError(new Error('Upload failed'));
      }
    } catch (err) {
      showError(err.message || t('上传失败'));
      onError(err);
    }
  };

  const loadPrompt = async () => {
    setLoading(true);
    try {
      let res = await API.get(`/api/prompt/${props.editingPrompt.id}`);
      const { success, message, data } = res.data;
      if (success) {
        const values = {
          ...data,
          status: data.status === 1,
          media_type: data.media_type || 'image',
          variables: data.variables || '',
          tags: data.tags ? JSON.parse(data.tags) : [],
        };
        // 解析 i18n
        let parsedI18n = {};
        if (data.i18n) {
          try {
            parsedI18n = JSON.parse(data.i18n);
          } catch {
            parsedI18n = {};
          }
        }
        // 兼容旧数据：如果 content_en 有值但 i18n.en 没有，同步到 i18n
        if (data.content_en && !parsedI18n.en?.content) {
          parsedI18n.en = { ...parsedI18n.en, content: data.content_en };
        }
        setI18nData(parsedI18n);
        formApiRef.current?.setValues({ ...getInitValues(), ...values });
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  useEffect(() => {
    if (formApiRef.current) {
      if (isEdit) {
        loadPrompt();
      } else {
        setI18nData({});
        setActiveLang(DEFAULT_LANG);
        formApiRef.current.setValues(getInitValues());
      }
    }
  }, [props.editingPrompt.id]);

  const handleAutoTranslate = async () => {
    const values = formApiRef.current?.getValues();
    const items = [];
    if (values.title?.trim()) items.push({ key: 'title', text: values.title.trim() });
    if (values.content?.trim()) items.push({ key: 'content', text: values.content.trim() });
    if (values.description?.trim()) items.push({ key: 'description', text: values.description.trim() });

    if (items.length === 0) {
      showError(t('请先填写中文内容'));
      return;
    }

    const targetLangs = ['EN', 'FR', 'RU', 'JA', 'VI', 'ZH-TW', 'ES', 'DE', 'KO', 'PT', 'IT'];
    setTranslating(true);

    for (let i = 0; i < targetLangs.length; i++) {
      const lang = targetLangs[i];
      setTranslateProgress(`${i + 1}/${targetLangs.length} ${lang}`);
      try {
        const res = await API.post('/api/translate/batch', {
          items,
          source_lang: 'ZH',
          target_langs: [lang],
        });
        if (res.data.success) {
          const newI18n = { ...i18nData };
          Object.entries(res.data.data).forEach(([l, fields]) => {
            const langKey = l.toLowerCase();
            newI18n[langKey] = { ...newI18n[langKey], ...fields };
          });
          setI18nData(newI18n);
          if (newI18n.en?.content) {
            formApiRef.current?.setValue('content_en', newI18n.en.content);
          }
        } else {
          console.warn('翻译失败:', lang, res.data.message);
        }
      } catch (err) {
        console.warn('翻译请求失败:', lang, err.message);
      }
    }

    setTranslating(false);
    setTranslateProgress('');
    showSuccess(t('翻译完成，已填充到各语言 Tab'));
  };

  const getLangField = (field) => {
    if (activeLang === DEFAULT_LANG) {
      return formApiRef.current?.getValue(field) || '';
    }
    return i18nData[activeLang]?.[field] || '';
  };

  const setLangField = (field, value) => {
    if (activeLang === DEFAULT_LANG) {
      formApiRef.current?.setValue(field, value);
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

  const hasTranslation = (code) => {
    if (code === DEFAULT_LANG) return true;
    const d = i18nData[code];
    return d && (d.title || d.content || d.description);
  };

  const submit = async (values) => {
    setLoading(true);
    let localInputs = { ...values };

    // Convert status boolean to number
    localInputs.status = localInputs.status ? 1 : 2;

    // Validate JSON fields but keep them as strings for the backend
    if (localInputs.variables && localInputs.variables.trim() !== '') {
      try {
        JSON.parse(localInputs.variables);
      } catch (e) {
        showError(t('变量格式不正确，请输入合法的JSON'));
        setLoading(false);
        return;
      }
    } else {
      localInputs.variables = '';
    }

    // Convert tags array to JSON string for backend
    localInputs.tags = JSON.stringify(localInputs.tags || []);

    localInputs.sort_order = parseInt(localInputs.sort_order) || 0;

    // 同步 i18n：en.content 同步到 content_en
    const finalI18n = { ...i18nData };
    if (finalI18n.en?.content) {
      localInputs.content_en = finalI18n.en.content;
    }
    localInputs.i18n = JSON.stringify(finalI18n);

    let res;
    try {
      if (isEdit) {
        res = await API.put(`/api/prompt/`, {
          ...localInputs,
          id: parseInt(props.editingPrompt.id),
        });
      } else {
        res = await API.post(`/api/prompt/`, {
          ...localInputs,
        });
      }
      const { success, message } = res.data;
      if (success) {
        if (isEdit) {
          showSuccess(t('提示词更新成功！'));
        } else {
          showSuccess(t('提示词创建成功！'));
        }
        props.refresh();
        props.handleClose();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  const renderLangFields = () => {
    return (
      <>
        {/* 默认语言字段 - 始终渲染但可能隐藏，避免 Form 状态丢失 */}
        <div style={{ display: activeLang === DEFAULT_LANG ? 'block' : 'none' }}>
          <Form.Input
            field='title'
            label={t('标题')}
            placeholder={t('请输入标题')}
            style={{ width: '100%' }}
            rules={[{ required: true, message: t('请输入标题') }]}
            showClear
          />
          <Form.TextArea
            field='content'
            label={t('内容')}
            placeholder={t('请输入提示词内容')}
            rows={4}
            style={{ width: '100%' }}
          />
          <Form.TextArea
            field='description'
            label={t('描述')}
            placeholder={t('请输入描述')}
            rows={2}
            style={{ width: '100%' }}
          />
        </div>

        {/* 非默认语言 - 受控组件，不注册到 Form */}
        {LANGUAGES.filter(l => l.code !== DEFAULT_LANG).map((lang) => (
          <div
            key={lang.code}
            style={{ display: activeLang === lang.code ? 'block' : 'none' }}
          >
            <div style={{ marginBottom: 16 }}>
              <label
                style={{
                  display: 'block',
                  marginBottom: 4,
                  fontSize: 14,
                  fontWeight: 600,
                  color: 'var(--semi-color-text-0)',
                }}
              >
                {t('标题')}
              </label>
              <Input
                value={i18nData[lang.code]?.title || ''}
                onChange={(v) => {
                  setI18nData((prev) => ({
                    ...prev,
                    [lang.code]: {
                      ...prev[lang.code],
                      title: v,
                    },
                  }));
                }}
                placeholder={t('请输入标题')}
                style={{ width: '100%' }}
              />
            </div>
            <div style={{ marginBottom: 16 }}>
              <label
                style={{
                  display: 'block',
                  marginBottom: 4,
                  fontSize: 14,
                  fontWeight: 600,
                  color: 'var(--semi-color-text-0)',
                }}
              >
                {t('内容')}
              </label>
              <Input.TextArea
                value={i18nData[lang.code]?.content || ''}
                onChange={(v) => {
                  setI18nData((prev) => ({
                    ...prev,
                    [lang.code]: {
                      ...prev[lang.code],
                      content: v,
                    },
                  }));
                }}
                rows={4}
                placeholder={t('请输入提示词内容')}
                style={{ width: '100%' }}
              />
            </div>
            <div style={{ marginBottom: 16 }}>
              <label
                style={{
                  display: 'block',
                  marginBottom: 4,
                  fontSize: 14,
                  fontWeight: 600,
                  color: 'var(--semi-color-text-0)',
                }}
              >
                {t('描述')}
              </label>
              <Input.TextArea
                value={i18nData[lang.code]?.description || ''}
                onChange={(v) => {
                  setI18nData((prev) => ({
                    ...prev,
                    [lang.code]: {
                      ...prev[lang.code],
                      description: v,
                    },
                  }));
                }}
                rows={2}
                placeholder={t('请输入描述')}
                style={{ width: '100%' }}
              />
            </div>
          </div>
        ))}
      </>
    );
  };

  return (
    <>
      <SideSheet
        placement={isEdit ? 'right' : 'left'}
        title={
          <Space>
            {isEdit ? (
              <Tag color='blue' shape='circle'>
                {t('更新')}
              </Tag>
            ) : (
              <Tag color='green' shape='circle'>
                {t('新建')}
              </Tag>
            )}
            <Title heading={4} className='m-0'>
              {isEdit ? t('更新提示词信息') : t('创建新的提示词')}
            </Title>
          </Space>
        }
        bodyStyle={{ padding: '0' }}
        visible={props.visiable}
        width={isMobile ? '100%' : 720}
        footer={
          <div className='flex justify-end bg-white'>
            <Space>
              <Button
                theme='solid'
                onClick={() => formApiRef.current?.submitForm()}
                icon={<IconSave />}
                loading={loading}
              >
                {t('提交')}
              </Button>
              <Button
                theme='light'
                type='primary'
                onClick={handleCancel}
                icon={<IconClose />}
              >
                {t('取消')}
              </Button>
            </Space>
          </div>
        }
        closeIcon={null}
        onCancel={() => handleCancel()}
      >
        <Spin spinning={loading}>
          <Form
            initValues={getInitValues()}
            getFormApi={(api) => (formApiRef.current = api)}
            onSubmit={submit}
          >
            {() => (
              <div className='p-2'>
                <Card className='!rounded-2xl shadow-sm border-0 mb-6'>
                  {/* Header: Basic Info */}
                  <div className='flex items-center mb-2'>
                    <Avatar
                      size='small'
                      color='blue'
                      className='mr-2 shadow-md'
                    >
                      <IconBookStroked size={16} />
                    </Avatar>
                    <div>
                      <Text className='text-lg font-medium'>
                        {t('基本信息')}
                      </Text>
                      <div className='text-xs text-gray-600'>
                        {t('设置提示词的基本信息')}
                      </div>
                    </div>
                  </div>

                  <Row gutter={12}>
                    <Col span={24}>
                      <Form.Input
                        field='cover_image_url'
                        label={t('封面图')}
                        placeholder={t('请输入封面图片地址或点击上传')}
                        style={{ width: '100%' }}
                        showClear
                        suffix={
                          <Upload
                            customRequest={handleCoverUpload}
                            accept='image/*'
                            showUploadList={false}
                            limit={1}
                          >
                            <Button
                              icon={<IconUpload size={14} />}
                              type='tertiary'
                              size='small'
                            >
                              {t('上传')}
                            </Button>
                          </Upload>
                        }
                      />
                    </Col>
                    <Col span={12}>
                      <Form.Input
                        field='author'
                        label={t('来源')}
                        placeholder={t('如 @username')}
                        style={{ width: '100%' }}
                        showClear
                      />
                    </Col>
                    <Col span={12}>
                      <Form.Input
                        field='model'
                        label={t('模型')}
                        placeholder={t('如 ChatGPT、Midjourney')}
                        style={{ width: '100%' }}
                        showClear
                      />
                    </Col>
                  </Row>

                  {/* 多语言 Tabs */}
                  <div style={{ marginTop: 16 }}>
                    <div className='flex justify-between items-center mb-2'>
                      <Text strong>{t('多语言内容')}</Text>
                      <Button
                        type='tertiary'
                        size='small'
                        icon={<IconLanguage />}
                        onClick={handleAutoTranslate}
                        loading={translating}
                        disabled={translating}
                      >
                        {translating ? `${t('翻译中')} ${translateProgress}` : `${t('自动翻译')}（AI）`}
                      </Button>
                    </div>
                    <Tabs
                      type='card'
                      activeKey={activeLang}
                      onChange={setActiveLang}
                      style={{ marginBottom: 12 }}
                      tabList={LANGUAGES.map((lang) => ({
                        tab: (
                          <span>
                            {lang.label}
                            {hasTranslation(lang.code) && lang.code !== DEFAULT_LANG && (
                              <span style={{ marginLeft: 4, color: 'var(--semi-color-success)' }}>●</span>
                            )}
                          </span>
                        ),
                        itemKey: lang.code,
                      }))}
                    />
                    {renderLangFields()}
                  </div>

                  <Row gutter={12}>
                    <Col span={12}>
                      <Form.Select
                        field='category_id'
                        label={t('分类')}
                        placeholder={t('请选择分类')}
                        style={{ width: '100%' }}
                        rules={[
                          { required: true, message: t('请选择分类') },
                        ]}
                      >
                        {props.categories?.map((cat) => (
                          <Form.Select.Option key={cat.id} value={cat.id}>
                            {cat.name}
                          </Form.Select.Option>
                        ))}
                      </Form.Select>
                    </Col>
                    <Col span={12}>
                      <Form.RadioGroup
                        field='media_type'
                        label={t('内容类型')}
                        type='button'
                        defaultValue='image'
                      >
                        <Form.Radio value='image'>{t('图片')}</Form.Radio>
                        <Form.Radio value='video'>{t('视频')}</Form.Radio>
                      </Form.RadioGroup>
                    </Col>
                  </Row>
                </Card>

                <Card className='!rounded-2xl shadow-sm border-0 mb-6'>
                  {/* Header: Advanced Settings */}
                  <div className='flex items-center mb-2'>
                    <Avatar
                      size='small'
                      color='green'
                      className='mr-2 shadow-md'
                    >
                      <IconBookStroked size={16} />
                    </Avatar>
                    <div>
                      <Text className='text-lg font-medium'>
                        {t('高级设置')}
                      </Text>
                      <div className='text-xs text-gray-600'>
                        {t('设置提示词的高级属性')}
                      </div>
                    </div>
                  </div>

                  <Row gutter={12}>
                    <Col span={24}>
                      <Form.TextArea
                        field='variables'
                        label={t('变量')}
                        placeholder={t('JSON格式 [{"name":"subject","label":"主题"}]')}
                        rows={2}
                        style={{ width: '100%' }}
                        rules={[
                          {
                            validator: (rule, value) => {
                              if (!value || value.trim() === '') {
                                return Promise.resolve();
                              }
                              return verifyJSONPromise(value);
                            },
                            message: t('请输入合法的JSON格式'),
                          },
                        ]}
                      />
                    </Col>
                    <Col span={24}>
                      <Form.TagInput
                        field='tags'
                        label={t('标签')}
                        placeholder={t('输入标签按回车添加')}
                        separator={','}
                        style={{ width: '100%' }}
                      />
                      <div className='flex flex-wrap gap-1 mt-2'>
                        {PRESET_TAGS.map((tag) => (
                          <Tag
                            key={tag}
                            size='small'
                            style={{ cursor: 'pointer' }}
                            onClick={() => {
                              const current =
                                formApiRef.current?.getValue('tags') || [];
                              if (!current.includes(tag)) {
                                formApiRef.current?.setValue('tags', [
                                  ...current,
                                  tag,
                                ]);
                              }
                            }}
                          >
                            {tag}
                          </Tag>
                        ))}
                      </div>
                    </Col>
                    <Col span={12}>
                      <Form.Input
                        field='sort_order'
                        label={t('排序')}
                        placeholder={t('请输入排序值')}
                        style={{ width: '100%' }}
                      />
                    </Col>
                    <Col span={12}>
                      <div className='flex items-center h-full pt-6'>
                        <Form.Switch
                          field='status'
                          label={t('状态')}
                          checkedText={t('启用')}
                          uncheckedText={t('禁用')}
                        />
                      </div>
                    </Col>
                  </Row>
                </Card>
              </div>
            )}
          </Form>
        </Spin>
      </SideSheet>
    </>
  );
};

export default EditPromptModal;
