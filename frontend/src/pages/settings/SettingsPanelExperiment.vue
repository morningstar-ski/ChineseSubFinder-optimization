<template>
  <div>
    <q-list style="max-width: 600px" dense>
      <q-item>
        <q-item-section>
          <q-item-label>自动转换字幕文件编码</q-item-label>
          <q-item-label caption>自动转换到目标编码，如果不是特殊情况，不建议开启，仅对新下载字幕生效</q-item-label>
          <q-item v-if="form.auto_change_sub_encode.enable">
            <q-item-section avatar top>
              <q-radio
                v-for="(v, k) in DESC_ENCODE_TYPE_NAME_MAP"
                :key="k"
                :label="v"
                v-model="form.auto_change_sub_encode.des_encode_type"
                :val="~~k"
              />
            </q-item-section>
          </q-item>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.auto_change_sub_encode.enable" />
        </q-item-section>
      </q-item>

      <q-separator spaced inset></q-separator>

      <q-item tag="label" :disable="!isChsChtChangerEnable" v-ripple>
        <q-item-section>
          <q-item-label>简、繁字幕互转功能</q-item-label>
          <q-item-label caption
            >需要开启"自动转换字幕文件编码"功能，并设置为转码"UTF-8"，否则无法启用和生效</q-item-label
          >
          <q-item v-if="form.chs_cht_changer.enable">
            <q-item-section avatar top>
              <q-radio
                :disable="!isChsChtChangerEnable"
                v-for="(v, k) in AUTO_CONVERT_LANG_NAME_MAP"
                :key="k"
                :label="v"
                v-model="form.chs_cht_changer.des_chinese_language_type"
                :val="~~k"
              />
            </q-item-section>
          </q-item>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle :disable="!isChsChtChangerEnable" v-model="form.chs_cht_changer.enable" />
        </q-item-section>
      </q-item>

      <q-separator spaced inset></q-separator>

      <q-item>
        <q-item-section>
          <q-item-label>远程Chrome</q-item-label>
          <q-item-label caption>
            本功能能够将本程序使用的 Chrome 操作移到一个有算力和资源的硬件上，这样部署本程序的资源要求进一步降低。<br />
            需要自行参看<a
              class="text-primary"
              href="https://go-rod.github.io/i18n/zh-CN/#/custom-launch?id=远程管理启动器"
              target="_blank"
              >https://go-rod.github.io/i18n/zh-CN/#/custom-launch?id=远程管理启动器</a
            >文档部署实验性功能，可用性和稳定性存疑，未必会继续支持更新。除非 go-rod 更新。
          </q-item-label>
        </q-item-section>
        <q-item-section avatar>
          <q-toggle v-model="form.remote_chrome_settings.enable" />
        </q-item-section>
      </q-item>

      <template v-if="form.remote_chrome_settings.enable">
        <q-item>
          <q-item-section>
            <q-item-label>Remote Chrome ws URL</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input
              v-model="form.remote_chrome_settings.remote_docker_url"
              placeholder="ws://192.168.xx.xx:9222"
              standout
              dense
              :rules="[(val) => (form.remote_chrome_settings.enable && !!val) || '不能为空']"
            />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <q-item-label>Remote user data dir</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input
              v-model="form.remote_chrome_settings.remote_user_data_dir"
              placeholder="/mnt/share/tmp"
              standout
              dense
            />
          </q-item-section>
        </q-item>
      </template>

      <q-separator spaced inset />

      <q-item>
        <q-item-section>
          <q-item-label>本地Chrome（可选覆盖）</q-item-label>
          <q-item-label caption>
            标准版 Docker 镜像现在已内置 Chromium，程序会自动探测
            <code>/usr/bin/chromium</code>、<code>/usr/bin/chromium-browser</code>
            等常见路径，一般不需要开启这个开关，也不需要手填路径。只有你想强制指定另一个浏览器可执行文件时，才需要在这里填写。注意以下几点：
            <div>
              <ol>
                <li>
                  如果你自己额外挂载了浏览器，可填写容器内完整路径，比如
                  <code>/app/cache/Plugin/Chrome/chrome</code>
                </li>
                <li>如果是 Windows 用户，那么就是你 Chrome.exe 的完整路径</li>
                <li>Chrome 版本不要太低</li>
                <li>请确认指定的chrome和对应平台、CPU架构一致</li>
              </ol>
            </div>
          </q-item-label>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.local_chrome_settings.enabled" />
        </q-item-section>
      </q-item>

      <template v-if="form.local_chrome_settings.enabled">
        <q-item>
          <q-item-section>
            <q-input
              v-model="form.local_chrome_settings.local_chrome_exe_f_path"
              label="Chrome(.exe) 的完整路径（可留空自动探测）"
              placeholder="/usr/bin/chromium"
              standout
              dense
              hint="标准版 Docker 留空会自动使用容器内已安装的 Chromium"
              persistent-hint
            />
          </q-item-section>
        </q-item>
      </template>

      <q-separator spaced inset />

      <q-item>
        <q-item-section>
          <q-item-label>API key</q-item-label>
          <q-item-label caption>
            本程序提供一些接口给开发者使用，通过API key鉴权，具体参见
            <!-- eslint-disable -->
            <a
              href="https://github.com/ChineseSubFinder/ChineseSubFinder/blob/docs/DesignFile/ApiKey%E8%AE%BE%E8%AE%A1/ApiKey%E8%AE%BE%E8%AE%A1.md"
              class="text-primary"
              target="_blank"
              >开发文档</a
            >
          </q-item-label>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.api_key_settings.enabled" />
        </q-item-section>
      </q-item>

      <template v-if="form.api_key_settings.enabled">
        <q-item>
          <q-btn label="重新生成密钥" color="primary" size="sm" @click="generateApiKey" />
        </q-item>
        <q-item class="q-mt-sm">
          <q-item-section>
            <q-input
              v-model="form.api_key_settings.key"
              standout
              dense
              :rules="[(val) => !!val || '不能为空']"
              readonly
            >
              <template #append>
                <copy-to-clipboard-btn v-if="form.api_key_settings.key" :text="form.api_key_settings.key" size="sm" />
              </template>
            </q-input>
          </q-item-section>
        </q-item>
      </template>
    </q-list>
  </div>
</template>

<script setup>
import { formModel } from 'pages/settings/use-settings';
import { toRefs } from '@vueuse/core';
import {
  AUTO_CONVERT_LANG_NAME_MAP,
  DESC_ENCODE_TYPE_NAME_MAP,
  DESC_ENCODE_TYPE_UTF8,
} from 'src/constants/SettingConstants';
import { computed } from 'vue';
import CopyToClipboardBtn from 'components/CopyToClipboardBtn';

const { experimental_function: form } = toRefs(formModel);

const isChsChtChangerEnable = computed(
  () =>
    formModel.experimental_function.auto_change_sub_encode?.enable &&
    formModel.experimental_function.auto_change_sub_encode?.des_encode_type === DESC_ENCODE_TYPE_UTF8
);

const generateUuid = () =>
  'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    // eslint-disable-next-line no-bitwise
    const r = (Math.random() * 16) | 0;
    // eslint-disable-next-line no-bitwise
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });

const generateApiKey = () => {
  const uuid = generateUuid();
  formModel.experimental_function.api_key_settings.key = uuid;
};
</script>
