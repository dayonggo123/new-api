import { useState, useEffect, useCallback } from 'react';
import { API, showError, showSuccess, copy } from '../../helpers';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Input,
  Tag,
  Modal,
  Form,
  Empty,
  Spin,
} from '@douyinfe/semi-ui';
import { Search, Copy, BookOpen, Sparkles } from 'lucide-react';
import './style.css';

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

  return (
    <div className='prompt-gallery-page'>
      {/* Header */}
      <div className='gallery-header'>
        <div className='header-content'>
          <BookOpen size={32} />
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
          prefix={<Search size={16} />}
          placeholder={t('搜索提示词...')}
          value={keyword}
          onChange={(v) => setKeyword(v)}
          className='search-input'
        />
      </div>

      {/* Prompt Grid */}
      <div className='prompt-grid'>
        {loading ? (
          <div className='loading-wrap'>
            <Spin size='large' />
          </div>
        ) : prompts.length === 0 ? (
          <Empty title={t('暂无提示词')} />
        ) : (
          prompts.map((prompt) => (
            <div key={prompt.id} className='prompt-card'>
              <div className='card-header'>
                <h3>{prompt.title}</h3>
                <Tag size='small' color='light-blue'>
                  {getCategoryName(prompt.category_id)}
                </Tag>
              </div>
              <div className='card-content'>
                <p>{prompt.content.slice(0, 120)}{prompt.content.length > 120 ? '...' : ''}</p>
              </div>
              {prompt.description && (
                <p className='card-desc'>{prompt.description}</p>
              )}
              <div className='card-actions'>
                <Button
                  theme='light'
                  type='tertiary'
                  icon={<Copy size={14} />}
                  onClick={() => handleCopy(prompt.content)}
                >
                  {t('复制')}
                </Button>
                <Button
                  theme='solid'
                  icon={<Sparkles size={14} />}
                  onClick={() => openDetail(prompt)}
                >
                  {t('详情')}
                </Button>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Detail Modal */}
      <Modal
        title={selectedPrompt?.title}
        visible={showDetail}
        onCancel={closeDetail}
        footer={null}
        width={600}
      >
        {selectedPrompt && (
          <div className='detail-content'>
            <Tag color='light-blue'>
              {getCategoryName(selectedPrompt.category_id)}
            </Tag>
            {selectedPrompt.description && (
              <p className='detail-desc'>{selectedPrompt.description}</p>
            )}

            {/* Variables Input */}
            {parseVariables(selectedPrompt.variables).length > 0 && (
              <div className='variables-section'>
                <h4>{t('变量')}</h4>
                {parseVariables(selectedPrompt.variables).map((v) => (
                  <div key={v.name} className='variable-row'>
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
            )}

            {/* Preview */}
            <div className='preview-section'>
              <h4>{t('预览')}</h4>
              <pre className='preview-text'>
                {renderPromptContent(selectedPrompt)}
              </pre>
            </div>

            <Button
              theme='solid'
              block
              icon={<Copy size={14} />}
              onClick={() =>
                handleCopy(renderPromptContent(selectedPrompt))
              }
            >
              {t('复制完整提示词')}
            </Button>
          </div>
        )}
      </Modal>
    </div>
  );
}
