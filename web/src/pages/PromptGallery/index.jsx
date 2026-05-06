import { useState, useEffect, useCallback, useMemo } from 'react';
import { API, showError, showSuccess, copy } from '../../helpers';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Input,
  Tag,
  Empty,
  Spin,
} from '@douyinfe/semi-ui';
import {
  IconSearch,
  IconCopy,
  IconClose,
  IconLanguage,
  IconHeartStroked,
} from '@douyinfe/semi-icons';
import './style.css';
import SEO from '../../components/seo/SEO';
import { WebPageSchema } from '../../components/seo/SchemaOrg';

const FALLBACK_IMAGE = 'data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 width=%22400%22 height=%22300%22%3E%3Crect width=%22400%22 height=%22300%22 fill=%22%23f0f0f0%22/%3E%3Ctext x=%2250%25%22 y=%2250%25%22 dominant-baseline=%22middle%22 text-anchor=%22middle%22 fill=%22%23999%22 font-size=%2214%22%3E%E6%9A%82%E6%97%A0%E5%9B%BE%E7%89%87%3C/text%3E%3C/svg%3E';

export default function PromptGallery() {
  const { t } = useTranslation();
  const [prompts, setPrompts] = useState([]);
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(true);
  const [keyword, setKeyword] = useState('');
  const [activeCategory, setActiveCategory] = useState(0);
  const [activeModel, setActiveModel] = useState('全部');
  const [activeTag, setActiveTag] = useState('全部');
  const [showAllTags, setShowAllTags] = useState(false);
  const [selectedPrompt, setSelectedPrompt] = useState(null);
  const [showDetail, setShowDetail] = useState(false);
  const [variableValues, setVariableValues] = useState({});

  // Load categories
  const loadCategories = useCallback(async () => {
    try {
      const res = await API.get('/api/public/prompt-categories');
      if (res.data.success) {
        setCategories(res.data.data || []);
      }
    } catch (error) {
      showError(error.message);
    }
  }, []);

  // Load prompts
  const loadPrompts = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (keyword) params.append('keyword', keyword);
      if (activeCategory > 0) params.append('category_id', activeCategory);
      params.append('p', '1');
      params.append('page_size', '100');

      const res = await API.get(`/api/public/prompts?${params.toString()}`);
      if (res.data.success) {
        setPrompts(res.data.data.items || []);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  }, [keyword, activeCategory]);

  useEffect(() => {
    loadCategories();
  }, [loadCategories]);

  useEffect(() => {
    const timer = setTimeout(() => {
      loadPrompts();
    }, 300);
    return () => clearTimeout(timer);
  }, [loadPrompts]);

  // Extract unique models from prompts
  const modelList = useMemo(() => {
    const models = new Set();
    prompts.forEach((p) => {
      if (p.model) models.add(p.model);
    });
    return ['全部', ...Array.from(models)];
  }, [prompts]);

  // Extract unique tags from prompts
  const tagList = useMemo(() => {
    const tags = new Set();
    prompts.forEach((p) => {
      const tagArr = parseTags(p.tags);
      tagArr.forEach((tag) => tags.add(tag));
    });
    return ['全部', ...Array.from(tags)];
  }, [prompts]);

  // Frontend filter by model and tag
  const filteredPrompts = useMemo(() => {
    let result = prompts;
    if (activeModel !== '全部') {
      result = result.filter((p) => p.model === activeModel);
    }
    if (activeTag !== '全部') {
      result = result.filter((p) => {
        const tagArr = parseTags(p.tags);
        return tagArr.includes(activeTag);
      });
    }
    return result;
  }, [prompts, activeModel, activeTag]);

  const handleCopy = async (text) => {
    if (await copy(text)) {
      showSuccess(t('已复制到剪贴板！'));
    }
  };

  const getCategoryName = (categoryId) => {
    const cat = categories.find((c) => c.id === categoryId);
    return cat ? cat.name : t('未分类');
  };

  const parseVariables = (variablesStr) => {
    if (!variablesStr) return [];
    try {
      return JSON.parse(variablesStr);
    } catch {
      return [];
    }
  };

  const parseTags = (tagsStr) => {
    if (!tagsStr) return [];
    try {
      return JSON.parse(tagsStr);
    } catch {
      return [];
    }
  };

  const renderPromptContent = (prompt) => {
    const variables = parseVariables(prompt.variables);
    if (variables.length === 0) return prompt.content;

    let result = prompt.content;
    variables.forEach((v) => {
      const val = variableValues[v.name] || v.example || '';
      result = result.replace(new RegExp(`\\{\\{${v.name}\\}\\}`, 'g'), val);
    });
    return result;
  };

  const openDetail = (prompt) => {
    setSelectedPrompt(prompt);
    setVariableValues({});
    setShowDetail(true);
  };

  const closeDetail = () => {
    setShowDetail(false);
    setSelectedPrompt(null);
  };

  const getEnglishContent = (prompt) => {
    return prompt.content_en || '';
  };

  const visibleTags = showAllTags ? tagList : tagList.slice(0, 13);

  return (
    <div className='prompt-gallery-page'>
      <SEO
        title={t('提示词画廊')}
        description={t('浏览和发现优质 AI 提示词，涵盖图像生成、文本创作、代码辅助等多个领域。')}
        pathname='/prompt-gallery'
        keywords='AI提示词, Prompt工程, 提示词模板, ChatGPT提示词, Midjourney提示词, AI创作'
      />
      <WebPageSchema
        pageTitle={t('提示词画廊')}
        pageDescription={t('浏览和发现优质 AI 提示词，涵盖图像生成、文本创作、代码辅助等多个领域。')}
        pathname='/prompt-gallery'
      />

      {/* Header */}
      <div className='gallery-header'>
        <div className='header-content'>
          <h1>OpenNana{t('提示词图库')}</h1>
          <p className='header-subtitle'>
            浏览和搜索GPT Image 2、Nano Banana 2/Pro、Seedance 2.0等AI图片、视频提示词案例，探索灵感。
          </p>

          {/* Search Bar */}
          <div className='header-search-bar'>
            <Input
              prefix={<IconSearch size={16} style={{ color: '#9ca3af' }} />}
              placeholder={t('搜索提示词...')}
              value={keyword}
              onChange={(v) => setKeyword(v)}
              className='header-search-input'
              onEnterPress={() => loadPrompts()}
            />
            <Button
              theme='solid'
              type='primary'
              className='header-search-btn'
              onClick={() => loadPrompts()}
            >
              {t('搜索')}
            </Button>
            <span className='header-search-count'>
              共 {filteredPrompts.length} 个{t('提示词')}
            </span>
          </div>

          {/* Model Filter */}
          {modelList.length > 1 && (
            <div className='header-filter-row'>
              <span className='header-filter-label'>模型：</span>
              <div className='header-filter-tags'>
                {modelList.map((model) => (
                  <span
                    key={model}
                    className={`header-filter-tag ${activeModel === model ? 'active' : ''}`}
                    onClick={() => setActiveModel(model)}
                  >
                    {model}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* Tag Filter */}
          {tagList.length > 1 && (
            <div className='header-filter-row'>
              <span className='header-filter-label' style={{ opacity: 0 }}>标签：</span>
              <div className='header-filter-tags'>
                {visibleTags.map((tag) => (
                  <span
                    key={tag}
                    className={`header-filter-tag ${activeTag === tag ? 'active' : ''}`}
                    onClick={() => setActiveTag(tag)}
                  >
                    {tag}
                  </span>
                ))}
                {tagList.length > 13 && (
                  <span
                    className='header-filter-tag more-btn'
                    onClick={() => setShowAllTags(!showAllTags)}
                  >
                    {showAllTags ? '收起 ▲' : '展开 ▼'}
                  </span>
                )}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Category Tabs (legacy, keep for compatibility) */}
      <div className='gallery-toolbar'>
        <div className='category-tabs'>
          <Tag
            className={activeCategory === 0 ? 'active' : ''}
            onClick={() => setActiveCategory(0)}
          >
            {t('全部分类')}
          </Tag>
          {categories.map((cat) => (
            <Tag
              key={cat.id}
              className={activeCategory === cat.id ? 'active' : ''}
              onClick={() => setActiveCategory(cat.id)}
            >
              {cat.name}
            </Tag>
          ))}
        </div>
      </div>

      {/* Masonry Grid */}
      <div className='gallery-masonry'>
        {loading ? (
          <div className='loading-wrap'>
            <Spin size='large' />
          </div>
        ) : filteredPrompts.length === 0 ? (
          <Empty title={t('暂无提示词')} />
        ) : (
          filteredPrompts.map((prompt) => (
            <div
              key={prompt.id}
              className='gallery-card'
              onClick={() => openDetail(prompt)}
            >
              <div className='gallery-card-image-wrap'>
                <img
                  src={prompt.cover_image_url || FALLBACK_IMAGE}
                  alt={prompt.title}
                  loading='lazy'
                  onError={(e) => {
                    e.target.src = FALLBACK_IMAGE;
                  }}
                />
                <div className='gallery-card-overlay'>
                  <span className='gallery-card-badge'>AI 生图</span>
                </div>
              </div>
              <div className='gallery-card-footer'>
                <h3>{prompt.title}</h3>
                <span className='gallery-card-category'>
                  {getCategoryName(prompt.category_id)}
                </span>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Detail Modal */}
      {showDetail && selectedPrompt && (
        <div className='gallery-detail-backdrop' onClick={closeDetail}>
          <div
            className='gallery-detail-modal'
            onClick={(e) => e.stopPropagation()}
          >
            {/* Modal Header */}
            <div className='detail-modal-header'>
              <div className='detail-modal-header-main'>
                <h2>{selectedPrompt.title}</h2>
                <div className='detail-meta-row'>
                  {selectedPrompt.author && (
                    <span className='detail-meta-item'>
                      {t('来源')}: {selectedPrompt.author}
                    </span>
                  )}
                  {selectedPrompt.model && (
                    <span className='detail-meta-item'>
                      {t('模型')}: {selectedPrompt.model}
                    </span>
                  )}
                </div>
                {parseTags(selectedPrompt.tags).length > 0 && (
                  <div className='detail-tags-row-compact'>
                    {parseTags(selectedPrompt.tags).map((tag, idx) => (
                      <Tag key={idx} size='small' color='light-blue'>
                        {tag}
                      </Tag>
                    ))}
                  </div>
                )}
              </div>
              <div className='detail-modal-actions'>
                <button
                  className='detail-modal-fav'
                  title={t('收藏')}
                  onClick={() => showSuccess(t('收藏功能即将上线'))}
                >
                  <IconHeartStroked size={18} />
                </button>
                <button className='detail-modal-close' onClick={closeDetail}>
                  <IconClose size={20} />
                </button>
              </div>
            </div>

            {/* Modal Body */}
            <div className='detail-modal-body'>
              {/* Cover Image */}
              <div className='detail-cover-image'>
                <img
                  src={selectedPrompt.cover_image_url || FALLBACK_IMAGE}
                  alt={selectedPrompt.title}
                  onError={(e) => {
                    e.target.src = FALLBACK_IMAGE;
                  }}
                />
              </div>

              {/* Variables Input */}
              {parseVariables(selectedPrompt.variables).length > 0 && (
                <div className='detail-variables-section'>
                  <h4 className='detail-section-title'>{t('变量')}</h4>
                  <div className='detail-variables-grid'>
                    {parseVariables(selectedPrompt.variables).map((v) => (
                      <div key={v.name} className='detail-variable-item'>
                        <label>{v.label || v.name}</label>
                        <Input
                          placeholder={v.example || ''}
                          value={variableValues[v.name] || ''}
                          onChange={(val) =>
                            setVariableValues((prev) => ({
                              ...prev,
                              [v.name]: val,
                            }))
                          }
                        />
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* English Prompt */}
              {getEnglishContent(selectedPrompt) && (
                <div className='detail-prompt-block'>
                  <div className='detail-prompt-header'>
                    <span className='detail-prompt-label'>
                      <IconLanguage size={14} />
                      English
                    </span>
                    <div className='detail-prompt-actions'>
                      <Button
                        theme='borderless'
                        size='small'
                        icon={<IconCopy size={12} />}
                        onClick={() =>
                          handleCopy(getEnglishContent(selectedPrompt))
                        }
                      >
                        {t('复制')}
                      </Button>
                    </div>
                  </div>
                  <pre className='detail-prompt-text'>
                    {getEnglishContent(selectedPrompt)}
                  </pre>
                </div>
              )}

              {/* Chinese Prompt */}
              <div className='detail-prompt-block'>
                <div className='detail-prompt-header'>
                  <span className='detail-prompt-label'>
                    <IconLanguage size={14} />
                    中文
                  </span>
                  <div className='detail-prompt-actions'>
                    <Button
                      theme='borderless'
                      size='small'
                      icon={<IconCopy size={12} />}
                      onClick={() =>
                        handleCopy(renderPromptContent(selectedPrompt))
                      }
                    >
                      {t('复制')}
                    </Button>
                  </div>
                </div>
                <pre className='detail-prompt-text'>
                  {renderPromptContent(selectedPrompt)}
                </pre>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
