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
  TextArea,
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

const EditPromptModal = (props) => {
  const { t } = useTranslation();
  const isEdit = props.editingPrompt.id !== undefined;
  const [loading, setLoading] = useState(isEdit);
  const [formValues, setFormValues] = useState(null);
  const [translating, setTranslating] = useState(false);
  const [translateProgress, setTranslateProgress] = useState({ current: 0, total: 0 });
  const [activeLang, setActiveLang] = useState(DEFAULT_LANG);
  const [i18nData, setI18nData] = useState({});
  const [titleI18nData, setTitleI18nData] = useState({});
  const [mediaType, setMediaType] = useState('image');
  const isMobile = useIsMobile();
  const formApiRef = useRef(null);
  const pollRef = useRef(null);

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
    video_url: '',
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
    formData.append('file', fileInstance);
    formData.append('media_type', 'cover_image');
    try {
      const res = await API.post('/api/prompt-media', formData);
      if (res.data.url) {
        onSuccess(res.data);
        formApiRef.current?.setValue('cover_image_url', res.data.url);
        showSuccess(t('封面上传成功'));
      } else {
        onError(new Error('Upload failed'));
      }
    } catch (err) {
      showError(err.message || t('上传失败'));
      onError(err);
    }
  };

  const extractVideoFirstFrame = (videoFile) => {
    return new Promise((resolve, reject) => {
      const video = document.createElement('video');
      video.preload = 'metadata';
      video.muted = true;
      video.playsInline = true;
      video.crossOrigin = 'anonymous';
      const url = URL.createObjectURL(videoFile);
      video.src = url;
      const cleanup = () => {
        URL.revokeObjectURL(url);
        video.remove();
      };
      video.addEventListener('loadeddata', () => {
        video.currentTime = 0.1;
      });
      video.addEventListener('seeked', () => {
        const canvas = document.createElement('canvas');
        canvas.width = video.videoWidth || 640;
        canvas.height = video.videoHeight || 360;
        const ctx = canvas.getContext('2d');
        ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
        canvas.toBlob((blob) => {
          cleanup();
          if (blob) {
            const frameFile = new File([blob], 'cover.jpg', { type: 'image/jpeg' });
            resolve(frameFile);
          } else {
            reject(new Error('Failed to extract frame'));
          }
        }, 'image/jpeg', 0.92);
      });
      video.addEventListener('error', (e) => {
        cleanup();
        reject(e);
      });
    });
  };

  const handleVideoUpload = async ({ fileInstance, onSuccess, onError }) => {
    try {
      // 1. Upload video
      const videoFormData = new FormData();
      videoFormData.append('file', fileInstance);
      videoFormData.append('media_type', 'video');
      const videoRes = await API.post('/api/prompt-media', videoFormData);
      if (!videoRes.data.url) {
        onError(new Error('Upload failed'));
        return;
      }

      // 2. Extract first frame and upload as cover (best effort)
      try {
        const frameFile = await extractVideoFirstFrame(fileInstance);
        const coverFormData = new FormData();
        coverFormData.append('file', frameFile);
        coverFormData.append('media_type', 'cover_image');
        const coverRes = await API.post('/api/prompt-media', coverFormData);
        if (coverRes.data.url) {
          formApiRef.current?.setValue('cover_image_url', coverRes.data.url);
        }
      } catch (frameErr) {
        console.warn('Failed to extract video frame:', frameErr);
      }

      formApiRef.current?.setValue('video_url', videoRes.data.url);
      onSuccess(videoRes.data);
      showSuccess(t('视频上传成功'));
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
          video_url: data.video_url || '',
          tags: data.tags ? JSON.parse(data.tags) : [],
        };
        setMediaType(values.media_type);
        let parsed = {};
        try { if (data.i18n) parsed = JSON.parse(data.i18n); } catch (e) {}
        setI18nData(parsed);
        let parsedTitle = {};
        try { if (data.title_i18n) parsedTitle = JSON.parse(data.title_i18n); } catch (e) {}
        setTitleI18nData(parsedTitle);
        setActiveLang(DEFAULT_LANG);
        formApiRef.current?.setValues({ ...getInitValues(), ...values });
        setFormValues({ ...getInitValues(), ...values });
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
        setMediaType('image');
        formApiRef.current.setValues(getInitValues());
        setFormValues(getInitValues());
        setI18nData({});
        setActiveLang(DEFAULT_LANG);
      }
    }
  }, [props.editingPrompt.id]);

  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  const handleAutoTranslate = async () => {
    const currentContent = formApiRef.current?.getValue('content');
    const currentTitle = formApiRef.current?.getValue('title');
    if (!currentContent || currentContent.trim() === '') {
      showError(t('请先填写内容'));
      return;
    }
    setTranslating(true);
    const targetLangs = LANGUAGES.filter((l) => {
      if (l.code === DEFAULT_LANG) return false;
      const hasContent = i18nData[l.code] && i18nData[l.code].trim() !== '';
      const hasTitle = titleI18nData[l.code] && titleI18nData[l.code].trim() !== '';
      return !hasContent || !hasTitle;
    }).map((l) => l.code);
    if (targetLangs.length === 0) {
      showSuccess(t('所有语言已翻译'));
      setTranslating(false);
      return;
    }
    setTranslateProgress({ current: 0, total: targetLangs.length });
    const failedLangs = [];
    try {
      const items = [{ key: 'content', text: currentContent.trim() }];
      if (currentTitle && currentTitle.trim() !== '') {
        items.push({ key: 'title', text: currentTitle.trim() });
      }
      const res = await API.post('/api/translate/queue', {
        items,
        source_lang: DEFAULT_LANG,
        target_langs: targetLangs,
      });
      if (!res.data.success) {
        showError(res.data.message || t('翻译失败'));
        setTranslating(false);
        return;
      }
      const queueId = res.data.data.queue_id;
      pollRef.current = setInterval(async () => {
        try {
          const pollRes = await API.get(`/api/translate/queue/${queueId}`);
          const queue = pollRes.data.data;
          if (!queue) return;
          // 更新翻译进度
          if (queue.progress) {
            setTranslateProgress({
              current: queue.progress.current || 0,
              total: queue.progress.total || targetLangs.length,
            });
          }
          if (queue.results) {
            Object.entries(queue.results).forEach(([langCode, result]) => {
              if (result && result.content) {
                // 检测翻译质量：跳过与原文相同的结果
                if (result.content.trim() === currentContent.trim()) {
                  failedLangs.push(langCode);
                  return;
                }
                setI18nData((prev) => ({ ...prev, [langCode]: result.content }));
                if (langCode === 'en') {
                  formApiRef.current?.setValue('content_en', result.content);
                }
              }
              if (result && result.title) {
                setTitleI18nData((prev) => ({ ...prev, [langCode]: result.title }));
              }
            });
          }
          if (queue.status === 'done') {
            if (pollRef.current) clearInterval(pollRef.current);
            pollRef.current = null;
            setTranslateProgress({ current: 0, total: 0 });
            if (failedLangs.length > 0) {
              const langLabels = failedLangs.map((code) => LANGUAGES.find((l) => l.code === code)?.label || code).join(', ');
              showError(t('以下语言翻译失败（结果与原文相同），请检查翻译 AI 配置：') + langLabels);
            } else {
              showSuccess(t('自动翻译完成'));
            }
            setTranslating(false);
          } else if (queue.status === 'failed') {
            if (pollRef.current) clearInterval(pollRef.current);
            pollRef.current = null;
            setTranslateProgress({ current: 0, total: 0 });
            showError(queue.error || t('翻译失败'));
            setTranslating(false);
          }
        } catch (err) {
          if (pollRef.current) clearInterval(pollRef.current);
          pollRef.current = null;
          setTranslateProgress({ current: 0, total: 0 });
          showError(err.message || t('翻译服务不可用'));
          setTranslating(false);
        }
      }, 2000);
    } catch (err) {
      showError(err.message || t('翻译服务不可用'));
      setTranslating(false);
    }
  };

  const handleRetranslate = async (targetLang) => {
    const currentContent = formApiRef.current?.getValue('content');
    const currentTitle = formApiRef.current?.getValue('title');
    if (!currentContent || currentContent.trim() === '') {
      showError(t('请先填写内容'));
      return;
    }
    setTranslating(true);
    try {
      const items = [{ key: 'content', text: currentContent.trim() }];
      if (currentTitle && currentTitle.trim() !== '') {
        items.push({ key: 'title', text: currentTitle.trim() });
      }
      const res = await API.post('/api/translate/batch', {
        items,
        source_lang: DEFAULT_LANG,
        target_langs: [targetLang],
      });
      const langResult = res.data.data?.[targetLang];
      if (res.data.success && langResult && langResult.content) {
        const translated = langResult.content;
        // 检测翻译质量：如果结果和原文相同，说明翻译失败
        if (translated.trim() === currentContent.trim()) {
          showError(t('翻译结果与原文相同，请检查翻译 AI 配置（模型是否支持多语言、API Key 是否有效）'));
          setTranslating(false);
          return;
        }
        setI18nData((prev) => ({ ...prev, [targetLang]: translated }));
        if (targetLang === 'en') {
          formApiRef.current?.setValue('content_en', translated);
        }
        if (langResult.title) {
          setTitleI18nData((prev) => ({ ...prev, [targetLang]: langResult.title }));
        }
        showSuccess(t('翻译完成'));
      } else {
        showError(t('翻译失败'));
      }
    } catch (err) {
      showError(err.message || t('翻译服务不可用'));
    }
    setTranslating(false);
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

    // Auto-fill content_en if empty (fallback to content)
    if (!localInputs.content_en || localInputs.content_en.trim() === '') {
      localInputs.content_en = localInputs.content || '';
    }

    // 多语言内容：content_en 从 i18nData 同步，i18n 排除英文（由 content_en 存储）
    const i18nForSave = { ...i18nData };
    delete i18nForSave.en; // 英文存 content_en
    delete i18nForSave.zh; // 中文存 content
    localInputs.i18n = JSON.stringify(i18nForSave);

    // 多语言标题：中文存 title，其他存 title_i18n
    const titleI18nForSave = { ...titleI18nData };
    delete titleI18nForSave.zh;
    localInputs.title_i18n = JSON.stringify(titleI18nForSave);

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
        width={isMobile ? '100%' : 600}
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
            key={isEdit ? `edit-${props.editingPrompt.id}` : 'new'}
            initValues={formValues || getInitValues()}
            getFormApi={(api) => (formApiRef.current = api)}
            onSubmit={submit}
            onValueChange={(values) => {
              if (values.media_type !== undefined && values.media_type !== mediaType) {
                setMediaType(values.media_type);
              }
            }}
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
                    {mediaType === 'video' && (
                      <Col span={24}>
                        <Form.Input
                          field='video_url'
                          label={t('视频文件')}
                          placeholder={t('请输入视频地址或点击上传')}
                          style={{ width: '100%' }}
                          showClear
                          suffix={
                            <Upload
                              customRequest={handleVideoUpload}
                              accept='video/*'
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
                    )}
                    <Col span={24}>
                      <Form.Input
                        field='title'
                        label={t('标题')}
                        placeholder={t('请输入标题')}
                        style={{ width: '100%' }}
                        rules={[
                          { required: true, message: t('请输入标题') },
                        ]}
                        showClear
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
                    <Col span={24}>
                      <div className='mb-2'>
                        <Text className='text-base font-medium'>{t('提示词内容')}</Text>
                      </div>
                      {/* 语言 Tab */}
                      <div style={{
                        display: 'flex',
                        flexWrap: 'wrap',
                        gap: 2,
                        borderBottom: '1px solid var(--semi-color-border)',
                        marginBottom: 12,
                      }}>
                        {LANGUAGES.map((lang) => {
                          const active = activeLang === lang.code;
                          return (
                            <button
                              key={lang.code}
                              type='button'
                              onClick={() => setActiveLang(lang.code)}
                              style={{
                                padding: '6px 12px',
                                border: 'none',
                                background: 'none',
                                cursor: 'pointer',
                                borderBottom: active ? '2px solid var(--semi-color-primary)' : '2px solid transparent',
                                color: active ? 'var(--semi-color-primary)' : 'var(--semi-color-text-2)',
                                fontWeight: active ? 600 : 400,
                                fontSize: 13,
                                transition: 'all 0.2s',
                                marginBottom: -1,
                              }}
                            >
                              {lang.label}
                            </button>
                          );
                        })}
                      </div>

                      {/* 中文 — 始终保留在 DOM 中 */}
                      <div style={{ display: activeLang === 'zh' ? 'block' : 'none' }}>
                        <Form.TextArea
                          field='content'
                          label={t('内容')}
                          placeholder={t('请输入提示词内容')}
                          rows={4}
                          style={{ width: '100%' }}
                          rules={[
                            { required: true, message: t('请输入内容') },
                          ]}
                        />
                      </div>

                      {/* 英文 — 始终保留在 DOM 中 */}
                      <div style={{ display: activeLang === 'en' ? 'block' : 'none' }}>
                        <div style={{ marginBottom: 8 }}>
                          <Button
                            type='tertiary'
                            size='small'
                            icon={<IconLanguage />}
                            loading={translating}
                            onClick={handleAutoTranslate}
                          >
                            {translateProgress.total > 0
                              ? `${t('自动翻译全部语言')} (${translateProgress.current}/${translateProgress.total})`
                              : t('自动翻译全部语言')}
                          </Button>
                        </div>
                        <div style={{ marginBottom: 12 }}>
                          <label style={{ display: 'block', marginBottom: 4, fontWeight: 500, color: 'var(--semi-color-text-0)' }}>
                            {t('标题')} (English)
                          </label>
                          <Input
                            value={titleI18nData['en'] || ''}
                            onChange={(v) => setTitleI18nData((prev) => ({ ...prev, en: v }))}
                            placeholder={t('请输入英文标题')}
                            style={{ width: '100%' }}
                          />
                        </div>
                        <Form.TextArea
                          field='content_en'
                          label={t('内容（英文）')}
                          placeholder={t('请输入英文提示词内容')}
                          rows={4}
                          style={{ width: '100%' }}
                        />
                      </div>

                      {/* 其他语言 — 始终保留在 DOM 中 */}
                      {LANGUAGES.filter(l => l.code !== 'zh' && l.code !== 'en').map((lang) => (
                        <div key={lang.code} style={{ display: activeLang === lang.code ? 'block' : 'none' }}>
                          <div style={{ marginBottom: 8 }}>
                            <Button
                              type='tertiary'
                              size='small'
                              icon={<IconLanguage />}
                              loading={translating}
                              onClick={() => handleRetranslate(lang.code)}
                            >
                              {t('重新翻译')}
                            </Button>
                          </div>
                          <div style={{ marginBottom: 12 }}>
                            <label style={{ display: 'block', marginBottom: 4, fontWeight: 500, color: 'var(--semi-color-text-0)' }}>
                              {t('标题')} ({lang.label})
                            </label>
                            <Input
                              value={titleI18nData[lang.code] || ''}
                              onChange={(v) => setTitleI18nData((prev) => ({ ...prev, [lang.code]: v }))}
                              placeholder={t('请输入翻译后的标题')}
                              style={{ width: '100%' }}
                            />
                          </div>
                          <div style={{ marginBottom: 12 }}>
                            <label style={{ display: 'block', marginBottom: 4, fontWeight: 500, color: 'var(--semi-color-text-0)' }}>
                              {t('内容')} ({lang.label})
                            </label>
                            <TextArea
                              value={i18nData[lang.code] || ''}
                              onChange={(v) => setI18nData((prev) => ({ ...prev, [lang.code]: v }))}
                              placeholder={t('请输入翻译后的提示词内容')}
                              rows={4}
                              style={{ width: '100%' }}
                            />
                          </div>
                        </div>
                      ))}
                    </Col>
                    <Col span={24}>
                      <Form.TextArea
                        field='description'
                        label={t('描述')}
                        placeholder={t('请输入描述')}
                        rows={2}
                        style={{ width: '100%' }}
                      />
                    </Col>
                    <Col span={12}>
                      <Form.Select
                        field='category_id'
                        label={t('分类')}
                        placeholder={t('请选择分类')}
                        style={{ width: '100%' }}
                        rules={[
                          { required: true, message: t('请选择分类') },
                        ]}
                        optionList={props.categories?.map((cat) => ({ label: cat.name, value: cat.id })) || []}
                      />
                    </Col>
                    <Col span={12}>
                      <Form.RadioGroup
                        field='media_type'
                        label={t('内容类型')}
                        type='button'
                        defaultValue='image'
                        options={[
                          { label: t('图片'), value: 'image' },
                          { label: t('视频'), value: 'video' },
                        ]}
                      />
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
                      <Form.InputNumber
                        field='sort_order'
                        label={t('排序')}
                        placeholder={t('请输入排序值')}
                        min={0}
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
