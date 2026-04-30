import { useState, useEffect, useCallback } from 'react';
import { API, showError, showSuccess, copy } from '../../helpers';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Input,
  Tag,
  Modal,
  Empty,
  Spin,
} from '@douyinfe/semi-ui';
import {
  IconSearch,
  IconCopy,
  IconBookStroked,
  IconClose,
  IconImage,
  IconLanguage,
  IconHeartStroked,
} from '@douyinfe/semi-icons';
import './style.css';

const FALLBACK_IMAGE = 'data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 width=%22400%22 height=%22300%22%3E%3Crect width=%22400%22 height=%22300%22 fill=%22%23f0f0f0%22/%3E%3Ctext x=%2250%25%22 y=%2250%25%22 dominant-baseline=%22middle%22 text-anchor=%22middle%22 fill=%22%23999%22 font-size=%2214%22%3E%E6%9A%82%E6%97%A0%E5%9B%BE%E7%89%87%3C/text%3E%3C/svg%3E';

export default function PromptGallery() {
  const { t } = useTranslation();
  const [prompts, setPrompts] = useState([]);
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(true);
  const [keyword, setKeyword] = useState('');
  const [activeCategory, setActiveCategory] = useState(0);
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

  // Use content_en if available, otherwise fallback
  const getEnglishContent = (prompt) => {
    return prompt.content_en || '';
  };

  return (
    <div className='prompt-gallery-page'>
      {/* Header */}
      <div className='gallery-header'>
        <div className='header-content'>
          <IconBookStroked size={36} style={{ opacity: 0.9 }} />
          <h1>{t('提示词画廊')}</h1>
          <p>{t('浏览和发现优质 AI 提示词')}</p>
        </div>
      </div>

      {/* Search & Filter */}
      <div className='gallery-toolbar'>
        <div className='category-tabs'>
          <Tag
            className={activeCategory === 0 ? 'active' : ''}
            onClick={() => setActiveCategory(0)}
          >
            {t('全部')}
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
        <Input
          prefix={<IconSearch size={16} />}
          placeholder={t('搜索提示词...')}
          value={keyword}
          onChange={(v) => setKeyword(v)}
          className='search-input'
        />
      </div>

      {/* Masonry Grid */}
      <div className='gallery-masonry'>
        {loading ? (
          <div className='loading-wrap'>
            <Spin size='large' />
          </div>
        ) : prompts.length === 0 ? (
          <Empty title={t('暂无提示词')} />
        ) : (
          prompts.map((prompt) => (
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
