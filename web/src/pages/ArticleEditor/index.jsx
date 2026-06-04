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
  Upload,
} from '@douyinfe/semi-ui';
import {
  IconSave,
  IconArrowLeft,
  IconUpload,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { useNavigate, useParams } from 'react-router-dom';
import { API, showError, showSuccess } from '../../helpers';
import HtmlRenderer from '../../components/common/HtmlRenderer';

// wangEditor
import '@wangeditor/editor/dist/css/style.css';
import { Editor, Toolbar } from '@wangeditor/editor-for-react';

const { Text, Title } = Typography;

/** 微信风格的工具栏配置 */
const toolbarConfig = {
  toolbarKeys: [
    'headerSelect',
    '|',
    'bold', 'italic', 'underline', 'through',
    'color', 'bgColor', 'fontSize', 'fontFamily',
    '|',
    'justifyLeft', 'justifyCenter', 'justifyRight', 'justifyJustify',
    'indent', 'delIndent',
    '|',
    'bulletedList', 'numberedList',
    '|',
    'quote', 'codeBlock', 'divider',
    '|',
    'insertLink', 'insertImage', 'uploadImage',
    'insertVideo', 'uploadVideo',
    '|',
    'insertTable',
    '|',
    'undo', 'redo',
    '|',
    'fullScreen',
  ],
};

const ArticleEditor = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { id } = useParams();
  const isEdit = !!id;

  const [loading, setLoading] = useState(isEdit);
  const [saving, setSaving] = useState(false);
  const [previewContent, setPreviewContent] = useState('');
  const [mediaType, setMediaType] = useState('image');
  const [categories, setCategories] = useState([]);
  const [editor, setEditor] = useState(null);
  const formApiRef = useRef(null);

  // wangEditor 配置
  const editorConfig = {
    placeholder: t('请输入正文内容，支持富文本格式、图片、视频等'),
    autoFocus: false,
    scroll: true,
    MENU_CONF: {
      uploadImage: {
        customUpload: async (file, insertFn) => {
          const formData = new FormData();
          formData.append('file', file);
          formData.append('media_type', 'content_image');
          try {
            const res = await API.post('/api/article-media', formData);
            if (res.data.url) {
              insertFn(res.data.url, file.name || '', res.data.url);
            } else {
              showError(t('图片上传失败'));
            }
          } catch (err) {
            showError(err.message || t('图片上传失败'));
          }
        },
      },
      uploadVideo: {
        customUpload: async (file, insertFn) => {
          const formData = new FormData();
          formData.append('file', file);
          formData.append('media_type', 'video');
          try {
            const res = await API.post('/api/article-media', formData);
            if (res.data.url) {
              insertFn(res.data.url);
            } else {
              showError(t('视频上传失败'));
            }
          } catch (err) {
            showError(err.message || t('视频上传失败'));
          }
        },
      },
    },
  };

  // Destroy editor on unmount
  useEffect(() => {
    return () => {
      if (editor == null) return;
      editor.destroy();
      setEditor(null);
    };
  }, [editor]);

  // Load content into editor when both editor and previewContent are ready
  useEffect(() => {
    if (editor && previewContent) {
      const current = editor.getHtml();
      if (current === '<p><br></p>' || current === '') {
        editor.setHtml(previewContent);
      }
    }
  }, [editor, previewContent]);

  // Load categories
  useEffect(() => {
    (async () => {
      try {
        const res = await API.get('/api/admin/article-categories');
        if (res.data.success) {
          setCategories(res.data.data || []);
        }
      } catch (err) {}
    })();
  }, []);

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

  // When editor is ready, load content for edit mode
  const handleEditorCreated = (editorInstance) => {
    setEditor(editorInstance);
    if (isEdit && previewContent) {
      editorInstance.setHtml(previewContent);
    }
  };

  const handleEditorChange = (editorInstance) => {
    const html = editorInstance.getHtml();
    formApiRef.current?.setValue('content', html);
    setPreviewContent(html);
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

  return (
    <Spin spinning={loading}>
      <div id="article-editor-container" style={{ maxWidth: 1400, margin: '0 auto', padding: '84px 24px 24px' }}>
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
                    <Form.Select field='category_id' label={t('分类')} optionList={categories.map((c) => ({ label: c.name, value: c.id }))} />
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

              {/* Content — wangEditor */}
              <Card className='!rounded-2xl shadow-sm border-0 mb-4'>
                <div style={{ display: 'flex', gap: 16 }}>
                  {/* Editor */}
                  <div style={{ flex: 1, border: '1px solid var(--semi-color-border)', borderRadius: 8, overflow: 'hidden' }}>
                    <Text type='tertiary' size='small' style={{ display: 'block', padding: '8px 12px', borderBottom: '1px solid var(--semi-color-border)', background: 'var(--semi-color-fill-0)' }}>
                      {t('正文内容')}
                    </Text>
                    <div style={{ display: 'flex', flexDirection: 'column', height: 560 }}>
                      <Toolbar
                        editor={editor}
                        defaultConfig={toolbarConfig}
                        mode="default"
                        style={{ borderBottom: '1px solid var(--semi-color-border)' }}
                      />
                      <Editor
                        defaultConfig={editorConfig}
                        defaultHtml={isEdit ? previewContent : '<p></p>'}
                        onCreated={handleEditorCreated}
                        onChange={handleEditorChange}
                        mode="default"
                        style={{ flex: 1, overflowY: 'auto' }}
                      />
                    </div>
                    {/* Hidden form field to hold content for validation */}
                    <Form.Input field='content' noLabel style={{ display: 'none' }} />
                  </div>

                  {/* Preview */}
                  <div style={{ flex: 1, border: '1px solid var(--semi-color-border)', borderRadius: 8, padding: 16, overflow: 'auto', maxHeight: 640 }}>
                    <Text type='tertiary' size='small' className='mb-2 block'>{t('实时预览')}</Text>
                    <HtmlRenderer content={previewContent || '<p></p>'} />
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
