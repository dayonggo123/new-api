import React, { useEffect, useState, useRef } from 'react';
import { Button, Col, Form, Row, Spin, Typography, Tag } from '@douyinfe/semi-ui';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

export default function SettingsSigninPoints(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    'signin_points_setting.enabled': true,
    'signin_points_setting.base_points': 10,
    'signin_points_setting.consecutive_bonus': '[0,1,2,2,2,5,5]',
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);

  function handleFieldChange(fieldName) {
    return (value) => {
      setInputs((inputs) => ({ ...inputs, [fieldName]: value }));
    };
  }

  function onSubmit() {
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const requestQueue = updateArray.map((item) => {
      let value = String(inputs[item.key]);
      return API.put('/api/option/', {
        key: item.key,
        value,
      });
    });
    setLoading(true);
    Promise.all(requestQueue)
      .then((res) => {
        if (requestQueue.length === 1) {
          if (res.includes(undefined)) return;
        } else if (requestQueue.length > 1) {
          if (res.includes(undefined))
            return showError(t('部分保存失败，请重试'));
        }
        showSuccess(t('保存成功'));
        props.refresh();
      })
      .catch(() => {
        showError(t('保存失败，请重试'));
      })
      .finally(() => {
        setLoading(false);
      });
  }

  useEffect(() => {
    const initialKeys = [
      'signin_points_setting.enabled',
      'signin_points_setting.base_points',
      'signin_points_setting.consecutive_bonus',
    ];
    const currentInputs = {};
    for (let key of initialKeys) {
      if (props.options.hasOwnProperty(key)) {
        let val = props.options[key];
        if (key === 'signin_points_setting.enabled') {
          val = val === true || val === 'true' || val === '1' || val === 1;
        } else if (key === 'signin_points_setting.base_points') {
          val = Number(val) || 0;
        }
        currentInputs[key] = val;
      }
    }
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    refForm.current.setValues(currentInputs);
  }, [props.options]);

  const parseBonus = (str) => {
    try {
      return JSON.parse(str || '[]');
    } catch {
      return [];
    }
  };

  const bonusArr = parseBonus(inputs['signin_points_setting.consecutive_bonus']);
  const dayLabels = ['第1天', '第2天', '第3天', '第4天', '第5天', '第6天', '第7天'];

  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('积分签到规则')}>
            <Typography.Text
              type='tertiary'
              style={{ marginBottom: 16, display: 'block' }}
            >
              {t('配置用户每日签到获得的积分规则')}
            </Typography.Text>
            <Row gutter={16}>
              <Col xs={24} sm={8} md={6} lg={6} xl={6}>
                <Form.Switch
                  field={'signin_points_setting.enabled'}
                  label={t('启用积分签到')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={handleFieldChange('signin_points_setting.enabled')}
                />
              </Col>
              <Col xs={24} sm={8} md={6} lg={6} xl={6}>
                <Form.InputNumber
                  field={'signin_points_setting.base_points'}
                  label={t('基础签到积分')}
                  placeholder={t('每日签到基础积分')}
                  onChange={handleFieldChange('signin_points_setting.base_points')}
                  min={0}
                  disabled={!inputs['signin_points_setting.enabled']}
                />
              </Col>
              <Col xs={24} sm={8} md={12} lg={12} xl={12}>
                <Form.TextArea
                  field={'signin_points_setting.consecutive_bonus'}
                  label={t('连续签到奖励（JSON数组）')}
                  placeholder='[0,1,2,2,2,5,5]'
                  onChange={handleFieldChange('signin_points_setting.consecutive_bonus')}
                  rows={2}
                  disabled={!inputs['signin_points_setting.enabled']}
                />
              </Col>
            </Row>
            <Row style={{ marginTop: 12, marginBottom: 16 }}>
              <Col span={24}>
                <Typography.Text type='secondary' size='small'>
                  {t('连续签到奖励预览：')}
                </Typography.Text>
                <div style={{ marginTop: 8, display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                  {dayLabels.map((label, idx) => {
                    const bonus = bonusArr[idx] ?? 0;
                    const total = (inputs['signin_points_setting.base_points'] || 0) + bonus;
                    return (
                      <Tag key={idx} color={bonus > 0 ? 'green' : 'grey'}>
                        {label}: {total}分
                        {bonus > 0 ? ` (+${bonus})` : ''}
                      </Tag>
                    );
                  })}
                </div>
              </Col>
            </Row>
            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存签到积分规则')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}
