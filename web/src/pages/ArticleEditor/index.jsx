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

import React, { useState, useEffect, useRef } from 'react';
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
  Select,
  Upload,
  Toast,
} from '@douyinfe/semi-ui';
import {
  IconSave,
  IconArrowLeft,
  IconUpload,
  IconBold,
  IconItalic,
  IconList,
  IconLink,
  IconCode,
  IconMinus,
  IconImage,
  IconVideo,
  IconQuote,
  IconH2,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { useNavigate, useParams } from 'react-router-dom';
import { API, showError, showSuccess } from '../../helpers';
import MarkdownRenderer from '../../components/common/markdown/MarkdownRenderer';

const { Text, Title } = Typography;

const ArticleEditor = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { id } = useParams();
  const isEdit = !!id;

  const [loading, setLoading] = useState(isEdit);
  const [saving, setSaving] = useState(false);
  const [previewContent, setPreviewContent] = useState('');
  const [mediaType, setMediaType] = useState('image');
  const formApiRef = useRef(null);
  const contentTextareaRef = useRef(null);

  // Load article for edit mode
  useEffect(() => {
    if (!isEdit) return;
    (async () => {
      try {
        const res = await API.get(`/api/admin/articles/${id}`);
        const { success, data } = res.data;
        if (success && data) {
          const values = {
            ...data,
            status: data.status === 1,
            is_featured: data.is_featured === true || data.is_featured === 1,
          };
          setMediaType(data.media_type || 'image');
          formApiRef.current?.setValues(values);
          setPreviewContent(data.content || '');
        } else {
          showError(t('文章不存在'));
          navigate('/console/article');
        }
      } catch (err) {
        showError(err.message);
        navigate('/console/article');
      }
      setLoading(false);
    })();
  }, [id]);

  // Insert markdown at cursor
  const getContentTextarea = () => {
    // Find the content field's textarea by data attribute or class
    return contentTextareaRef.current || document.querySelector('#article-editor-container textarea');
  };

  const insertMarkdown = (prefix, suffix = '', placeholder = '') => {
    const textarea = getContentTextarea();
    if (!textarea) return;
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const selected = textarea.value.substring(start, end);
    const before = textarea.value.substring(0, start);
    const after = textarea.value.substring(end);
    const inner = selected || placeholder;
    const newText = before + prefix + inner + suffix + after;
    formApiRef.current?.setValue('content', newText);
    // Restore cursor position
    setTimeout(() => {
      textarea.focus();
      const cursorPos = start + prefix.length;
      if (!selected) {
        textarea.setSelectionRange(cursorPos, cursorPos + placeholder.length);
      } else {
        textarea.setSelectionRange(cursorPos, cursorPos + selected.length);
      }
    }, 0);
  };

  // Upload and insert content image
  const handleUploadContentImage = () => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = 'image/*';
    input.onchange = async (e) => {
      const file = e.target.files?.[0];
      if (!file) return;
      const formData = new FormData();
      formData.append('file', file);
      formData.append('media_type', 'cover_image');
      try {
        const res = await API.post('/api/article-media', formData);
        if (res.data.url) {
          const alt = prompt(t('请输入图片描述'), file.name.split('.')[0]) || '';
          insertMarkdown(`![${alt}](${res.data.url})`);
        } else {
          showError(t('上传失败'));
        }
      } catch (err) {
        showError(err.message || t('上传失败'));
      }
    };
    input.click();
  };

  const handleInsertVideo = () => {
    const url = prompt(t('请输入视频 URL'));
    if (!url?.trim()) return;
    insertMarkdown(`\n<video controls src="${url.trim()}"></video>\n`, '', '');
  };

  // Submit
  const submit = async (values) => {
    setSaving(true);
    try {
      const payload = {
        ...values,
        status: values.status ? 1 : 2,
        is_featured: values.is_featured ? true : false,
        category_id: parseInt(values.category_id) || 0,
      };
      let res;
      if (isEdit) {
        res = await API.put(`/api/admin/articles/${id}`, { ...payload, id: parseInt(id) });
      } else {
        res = await API.post('/api/admin/articles', payload);
      }
      const { success, message } = res.data;
      if (success) {
        showSuccess(isEdit ? t('文章更新成功') : t('文章创建成功'));
        navigate('/console/article');
      } else {
        showError(message);
      }
    } catch (err) {
      showError(err.message);
    }
    setSaving(false);
  };

  const toolbarBtns = [
    { icon: <IconBold />, title: t('加粗'), action: () => insertMarkdown('**', '**', t('加粗文本')) },
    { icon: <IconItalic />, title: t('斜体'), action: () => insertMarkdown('*', '*', t('斜体文本')) },
    { icon: <IconH2 />, title: t('H2 标题'), action: () => insertMarkdown('## ', '', t('标题文本')) },
    { icon: <IconLink />, title: t('链接'), action: () => insertMarkdown('[', '](url)', t('链接文本')) },
    { icon: <IconList />, title: t('无序列表'), action: () => insertMarkdown('- ', '', t('列表项')) },
    { icon: <IconQuote />, title: t('引用'), action: () => insertMarkdown('> ', '', t('引用文本')) },
    { icon: <IconCode />, title: t('行内代码'), action: () => insertMarkdown('`', '`', t('代码')) },
    { icon: <IconCode />, title: t('代码块'), action: () => insertMarkdown('```\n', '\n```', t('在此输入代码')) },
    { icon: <IconMinus />, title: t('分割线'), action: () => insertMarkdown('\n---\n', '', '') },
    { type: 'divider' },
    { icon: <IconImage />, title: t('插入图片'), action: handleUploadContentImage },
    { icon: <IconVideo />, title: t('插入视频'), action: handleInsertVideo },
  ];

  return (
    <Spin spinning={loading}>
      <div id="article-editor-container" style={{ maxWidth: 1400, margin: '0 auto', padding: '24px' }}>
        {/* Header */}
        <div className='flex items-center justify-between mb-4'>
          <Space>
            <Button icon={<IconArrowLeft />} type='tertiary' onClick={() => navigate('/console/article')}>
              {t('返回')}
            </Button>
            <Tag color='blue' shape='circle'>{isEdit ? t('编辑') : t('新建')}</Tag>
            <Title heading={4} className='m-0'>{isEdit ? t('编辑文章') : t('新建文章')}</Title>
          </Space>
          <Button theme='solid' icon={<IconSave />} loading={saving} onClick={() => formApiRef.current?.submitForm()}>
            {t('保存')}
          </Button>
        </div>

        {/* Form */}
        <Form
          getFormApi={(api) => { formApiRef.current = api; }}
          onValueChange={(values) => { setPreviewContent(values.content || ''); }}
          onSubmit={submit}
          style={{ width: '100%' }}
        >
          {({ formState, formApi }) => (
            <div>
              {/* Basic Info */}
              <Card className='!rounded-2xl shadow-sm border-0 mb-4'>
                <Row gutter={12}>
                  <Col span={24}>
                    <Form.Input field='title' label={t('标题')} placeholder={t('请输入文章标题')} rules={[{ required: true, message: t('标题不能为空') }]} showClear />
                  </Col>
                  <Col span={12}>
                    <Form.Input field='slug' label={t('Slug')} placeholder={t('URL 友好标识，留空自动生成')} />
                  </Col>
                  <Col span={12}>
                    <Form.Select field='category_id' label={t('分类')} />
                  </Col>
                  <Col span={8}>
                    <Form.Input field='author' label={t('作者')} placeholder={t('作者名称')} />
                  </Col>
                  <Col span={8}>
                    <Form.Select field='media_type' label={t('封面类型')} optionList={[{ label: t('图片'), value: 'image' }, { label: t('视频'), value: 'video' }]} onChange={(v) => setMediaType(v)} />
                  </Col>
                  <Col span={8}>
                    {mediaType === 'video' ? (
                      <Form.Input field='video_url' label={t('视频 URL')} placeholder={t('视频封面')} showClear
                        suffix={
                          <Upload customRequest={({ fileInstance, onSuccess, onError }) => {
                            const fd = new FormData();
                            fd.append('file', fileInstance);
                            fd.append('media_type', 'video');
                            API.post('/api/article-media', fd).then(r => {
                              if (r.data.url) { onSuccess(r.data); formApi.setValue('video_url', r.data.url); showSuccess('上传成功'); }
                              else onError(new Error('fail'));
                            }).catch(e => { showError(e.message); onError(e); });
                          }} accept='video/*' showUploadList={false} limit={1}>
                            <Button icon={<IconUpload size={14} />} type='tertiary' size='small'>{t('上传')}</Button>
                          </Upload>
                        }
                      />
                    ) : (
                      <Form.Input field='cover_image_url' label={t('封面图 URL')} placeholder={t('封面图片地址')} showClear
                        suffix={
                          <Upload customRequest={({ fileInstance, onSuccess, onError }) => {
                            const fd = new FormData();
                            fd.append('file', fileInstance);
                            fd.append('media_type', 'cover_image');
                            API.post('/api/article-media', fd).then(r => {
                              if (r.data.url) { onSuccess(r.data); formApi.setValue('cover_image_url', r.data.url); showSuccess('封面上传成功'); }
                              else onError(new Error('fail'));
                            }).catch(e => { showError(e.message); onError(e); });
                          }} accept='image/*' showUploadList={false} limit={1}>
                            <Button icon={<IconUpload size={14} />} type='tertiary' size='small'>{t('上传')}</Button>
                          </Upload>
                        }
                      />
                    )}
                  </Col>
                  <Col span={12}>
                    <Form.Input field='tags' label={t('标签')} placeholder={t('JSON 数组，如 ["tag1"]')} />
                  </Col>
                  <Col span={6}>
                    <div className='flex items-center h-full pt-6'>
                      <Form.Switch field='status' label={t('状态')} checkedText={t('启用')} uncheckedText={t('禁用')} />
                    </div>
                  </Col>
                  <Col span={6}>
                    <div className='flex items-center h-full pt-6'>
                      <Form.Switch field='is_featured' label={t('精选')} checkedText={t('是')} uncheckedText={t('否')} />
                    </div>
                  </Col>
                  <Col span={24}>
                    <Form.TextArea field='summary' label={t('摘要')} placeholder={t('文章摘要')} rows={3} />
                  </Col>
                </Row>
              </Card>

              {/* Content */}
              <Card className='!rounded-2xl shadow-sm border-0 mb-4'>
                <div style={{ display: 'flex', gap: 16 }}>
                  <div style={{ flex: 1 }}>
                    <Text type='tertiary' size='small' className='mb-2 block'>{t('正文内容')}</Text>
                    {/* Toolbar */}
                    <div style={{ marginBottom: 8, display: 'flex', flexWrap: 'wrap', gap: 4, padding: '6px 8px', background: 'var(--semi-color-fill-0)', borderRadius: 6, border: '1px solid var(--semi-color-border)' }}>
                      {toolbarBtns.map((btn, i) =>
                        btn.type === 'divider' ? (
                          <div key={i} style={{ width: 1, height: 24, background: 'var(--semi-color-border)', margin: '0 4px' }} />
                        ) : (
                          <Button key={i} size='small' type='tertiary' icon={btn.icon} onClick={btn.action} title={btn.title} />
                        )
                      )}
                    </div>
                    <Form.TextArea field='content' placeholder={t('支持 Markdown 格式、图片、视频等')} rules={[{ required: true, message: t('内容不能为空') }]}
                      style={{ fontFamily: 'monospace', minHeight: 500 }} rows={22}
                    />
                  </div>
                  <div style={{ flex: 1, border: '1px solid var(--semi-color-border)', borderRadius: 8, padding: 16, overflow: 'auto', maxHeight: 640 }}>
                    <Text type='tertiary' size='small' className='mb-2 block'>{t('实时预览')}</Text>
                    <MarkdownRenderer content={previewContent || ''} />
                  </div>
                </div>
              </Card>
            </div>
          )}
        </Form>
      </div>
    </Spin>
  );
};

export default ArticleEditor;
