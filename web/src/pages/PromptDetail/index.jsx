import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button, Tag, Spin, Typography, Breadcrumb } from '@douyinfe/semi-ui';
import { IconCopy, IconArrowLeft } from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../helpers';
import SEO from '../../components/seo/SEO';
import { WebPageSchema, FAQPageSchema } from '../../components/seo/SchemaOrg';

const { Title, Text } = Typography;

const FALLBACK_IMAGE = 'data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 width=%22400%22 height=%22300%22%3E%3Crect width=%22400%22 height=%22300%22 fill=%22%23f0f0f0%22/%3E%3Ctext x=%2250%25%22 y=%2250%25%22 dominant-baseline=%22middle%22 text-anchor=%22middle%22 fill=%22%23999%22 font-size=%2214%22%3E%E6%9A%82%E6%97%A0%E5%9B%BE%E7%89%87%3C/text%3E%3C/svg%3E';

function parseTags(tagsStr) {
  if (!tagsStr) return [];
  try {
    return JSON.parse(tagsStr);
  } catch {
    return [];
  }
}

function parseFAQ(faqStr) {
  if (!faqStr) return [];
  try {
    return JSON.parse(faqStr);
  } catch {
    return [];
  }
}

export default function PromptDetail() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [prompt, setPrompt] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadPrompt();
  }, [id]);

  const loadPrompt = async () => {
    setLoading(true);
    try {
      const res = await API.get(`/api/public/prompts/${id}`);
      const { success, data, message } = res.data;
      if (success) {
        setPrompt(data);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
    setLoading(false);
  };

  const handleCopy = (text) => {
    navigator.clipboard.writeText(text).then(() => {
      showSuccess(t('已复制到剪贴板'));
    });
  };

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Spin size='large' tip={t('加载中...')} />
      </div>
    );
  }

  if (!prompt) {
    return (
      <div style={{ minHeight: '100vh', padding: '80px 20px', textAlign: 'center' }}>
        <Title heading={3}>{t('提示词不存在')}</Title>
        <Button icon={<IconArrowLeft />} onClick={() => navigate('/prompt-gallery')}>
          {t('返回画廊')}
        </Button>
      </div>
    );
  }

  const tags = parseTags(prompt.tags);
  const faqList = parseFAQ(prompt.faq);
  const description = prompt.intro || prompt.description || prompt.content?.slice(0, 200) || '';
  const keywords = prompt.seo_keywords || tags.join(', ');

  return (
    <div className='prompt-detail-page' style={{ minHeight: '100vh', background: 'var(--semi-color-bg-0)', paddingTop: 64 }}>
      <SEO
        title={prompt.title}
        description={description}
        pathname={`/prompt/${id}`}
        keywords={keywords}
        ogImage={prompt.cover_image_url}
        type='article'
      />
      <WebPageSchema
        pageTitle={prompt.title}
        pageDescription={description}
        pathname={`/prompt/${id}`}
      />
      {faqList.length > 0 && (
        <FAQPageSchema faqs={faqList} />
      )}

      <div style={{ maxWidth: 800, margin: '0 auto', padding: '24px 16px' }}>
        <Breadcrumb>
          <Breadcrumb.Item onClick={() => navigate('/prompt-gallery')}>
            {t('提示词画廊')}
          </Breadcrumb.Item>
          <Breadcrumb.Item>{prompt.title}</Breadcrumb.Item>
        </Breadcrumb>

        <div style={{ marginTop: 24 }}>
          {/* Title */}
          <Title heading={2} style={{ marginBottom: 12 }}>
            {prompt.title}
          </Title>

          {/* Meta */}
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, marginBottom: 16, color: 'var(--semi-color-text-2)', fontSize: 14 }}>
            {prompt.author && <span>{t('来源')}: {prompt.author}</span>}
            {prompt.model && <span>{t('模型')}: {prompt.model}</span>}
            <span>
              {t('类型')}: {prompt.media_type === 'video' ? t('视频') : t('图片')}
            </span>
          </div>

          {/* Tags */}
          {tags.length > 0 && (
            <div style={{ marginBottom: 20 }}>
              {tags.map((tag, idx) => (
                <Tag key={idx} color='light-blue' style={{ marginRight: 8, marginBottom: 8 }}>
                  {tag}
                </Tag>
              ))}
            </div>
          )}

          {/* Cover Image */}
          {prompt.cover_image_url && (
            <div style={{ marginBottom: 24 }}>
              <img
                src={prompt.cover_image_url || FALLBACK_IMAGE}
                alt={prompt.title}
                style={{ width: '100%', maxWidth: 640, borderRadius: 12, objectFit: 'cover' }}
                onError={(e) => { e.target.src = FALLBACK_IMAGE; }}
              />
            </div>
          )}

          {/* Intro */}
          {prompt.intro && (
            <div style={{ background: '#eef2ff', padding: 16, borderRadius: 8, borderLeft: '4px solid #4f46e5', marginBottom: 24 }}>
              <Text style={{ fontSize: 15, lineHeight: 1.6 }}>{prompt.intro}</Text>
            </div>
          )}

          {/* Content */}
          <div style={{ marginBottom: 24 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
              <Title heading={4} style={{ margin: 0 }}>{t('提示词内容')}</Title>
              <Button icon={<IconCopy />} size='small' onClick={() => handleCopy(prompt.content)}>
                {t('复制')}
              </Button>
            </div>
            <pre style={{ background: '#f8f9fa', padding: 16, borderRadius: 8, whiteSpace: 'pre-wrap', wordBreak: 'break-word', lineHeight: 1.6, fontSize: 14, border: '1px solid var(--semi-color-border)' }}>
              {prompt.content}
            </pre>
          </div>

          {/* English Content */}
          {prompt.content_en && (
            <div style={{ marginBottom: 24 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                <Title heading={4} style={{ margin: 0 }}>English Prompt</Title>
                <Button icon={<IconCopy />} size='small' onClick={() => handleCopy(prompt.content_en)}>
                  {t('复制')}
                </Button>
              </div>
              <pre style={{ background: '#f8f9fa', padding: 16, borderRadius: 8, whiteSpace: 'pre-wrap', wordBreak: 'break-word', lineHeight: 1.6, fontSize: 14, border: '1px solid var(--semi-color-border)' }}>
                {prompt.content_en}
              </pre>
            </div>
          )}

          {/* FAQ */}
          {faqList.length > 0 && (
            <div style={{ marginBottom: 24 }}>
              <Title heading={4} style={{ marginBottom: 16 }}>{t('常见问题')}</Title>
              {faqList.map((item, idx) => (
                <details key={idx} style={{ marginBottom: 12, padding: 12, background: '#f9fafb', borderRadius: 8, border: '1px solid var(--semi-color-border)' }}>
                  <summary style={{ fontWeight: 600, color: '#4f46e5', cursor: 'pointer', fontSize: 15 }}>
                    {item.question}
                  </summary>
                  <div style={{ marginTop: 8, color: '#555', lineHeight: 1.6 }}>
                    {item.answer}
                  </div>
                </details>
              ))}
            </div>
          )}

          {/* Back */}
          <Button icon={<IconArrowLeft />} onClick={() => navigate('/prompt-gallery')}>
            {t('返回画廊')}
        </Button>
        </div>
      </div>
    </div>
  );
}
