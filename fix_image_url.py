content = open(r'f:\new api\relay\channel\task\veo\adaptor.go').read()
old = '\tcase 2: // completed\n\t\ttaskResult.Status = model.TaskStatusSuccess\n\t\ttaskResult.Progress = "100%"\n\t\t// Extract video URL from generated_video array\n\t\tif len(h.GeneratedVideo) > 0 && h.GeneratedVideo[0].VideoURL != "" {\n\t\t\ttaskResult.Url = h.GeneratedVideo[0].VideoURL\n\t\t}\n\tcase 3: // failed'
new = '\tcase 2: // completed\n\t\ttaskResult.Status = model.TaskStatusSuccess\n\t\ttaskResult.Progress = "100%"\n\t\t// Extract video URL from generated_video array\n\t\tif len(h.GeneratedVideo) > 0 && h.GeneratedVideo[0].VideoURL != "" {\n\t\t\ttaskResult.Url = h.GeneratedVideo[0].VideoURL\n\t\t}\n\t\t// Extract image URL from generated_image array\n\t\tif len(h.GeneratedImage) > 0 and h.GeneratedImage[0].ImageURL != "" {\n\t\t\ttaskResult.Url = h.GeneratedImage[0].ImageURL\n\t\t}\n\tcase 3: // failed'
if old in content:
    content = content.replace(old, new)
    open(r'f:\new api\relay\channel\task\veo\adaptor.go','w').write(content)
    print('done')
else:
    # Try tabs as actual tab chars
    old2 = old.replace('\\t', '\t')
    new2 = new.replace('\\t', '\t')
    if old2 in content:
        content = content.replace(old2, new2)
        open(r'f:\new api\relay\channel\task\veo\adaptor.go','w').write(content)
        print('done with actual tabs')
    else:
        print('NOT FOUND')
        # Debug: print lines around case 2
        lines = content.split('\n')
        for i, l in enumerate(lines):
            if 'case 2' in l or 'VideoURL' in l:
                print(f"LINE {i+1}: {repr(l)}")
